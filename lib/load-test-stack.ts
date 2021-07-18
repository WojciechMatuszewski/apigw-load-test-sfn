import * as cdk from "@aws-cdk/core";
import * as apigw from "@aws-cdk/aws-apigatewayv2";
import * as apigwIntegrations from "@aws-cdk/aws-apigatewayv2-integrations";
import * as authorizers from "@aws-cdk/aws-apigatewayv2-authorizers";
import * as cognito from "@aws-cdk/aws-cognito";
import * as iam from "@aws-cdk/aws-iam";
import * as sfn from "@aws-cdk/aws-stepfunctions";
import * as sfnTasks from "@aws-cdk/aws-stepfunctions-tasks";
import { GoFunction, GoFunctionProps } from "@aws-cdk/aws-lambda-go";
import { join } from "path";

export class LoadTestStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const loadTestEndpointFunction = new LambdaGoFunction(
      this,
      "loadTestEndpointFunction",
      {
        name: "load-test-endpoint"
      }
    );

    const userPool = new cognito.UserPool(this, "userPool", {
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      signInAliases: {
        email: true
      },
      signInCaseSensitive: false,
      // Needed for the admin sign up
      selfSignUpEnabled: true,
      passwordPolicy: {
        minLength: 6,
        requireDigits: false,
        requireLowercase: false,
        requireSymbols: false,
        requireUppercase: false
      }
    });

    const userPoolClient = new cognito.UserPoolClient(this, "userPoolClient", {
      userPool,
      authFlows: {
        adminUserPassword: true
      },
      generateSecret: false
    });

    const apiAuthorizer = new authorizers.HttpUserPoolAuthorizer({
      userPool,
      userPoolClient
    });

    const api = new apigw.HttpApi(this, "api");
    api.addRoutes({
      path: "/",
      methods: [apigw.HttpMethod.GET],
      integration: new apigwIntegrations.LambdaProxyIntegration({
        handler: loadTestEndpointFunction
      }),
      authorizer: apiAuthorizer
    });

    const createUserIdsFunction = new LambdaGoFunction(this, "createUserIds", {
      name: "create-user-ids"
    });

    const createUserIdsTask = new sfnTasks.LambdaInvoke(
      this,
      "createUserIdsTask",
      {
        lambdaFunction: createUserIdsFunction,
        resultPath: "$",
        payloadResponseOnly: true
      }
    );

    const createUserFunction = new LambdaGoFunction(this, "createUser", {
      name: "create-user",
      environment: {
        USER_POOL_ID: userPool.userPoolId,
        USER_POOL_CLIENT_ID: userPoolClient.userPoolClientId
      }
    });
    const cognitoPolicy = iam.ManagedPolicy.fromAwsManagedPolicyName(
      "AmazonCognitoPowerUser"
    );
    const smPolicy = iam.ManagedPolicy.fromAwsManagedPolicyName(
      "SecretsManagerReadWrite"
    );
    createUserFunction.role?.addManagedPolicy(cognitoPolicy);
    createUserFunction.role?.addManagedPolicy(smPolicy);
    createUserFunction.addToRolePolicy(
      new iam.PolicyStatement({
        effect: iam.Effect.ALLOW,
        resources: ["*"],
        actions: ["ssm:*"]
      })
    );

    const createUserTask = new sfnTasks.LambdaInvoke(this, "createUserTask", {
      lambdaFunction: createUserFunction,
      payloadResponseOnly: true
    });

    const createUsersStep = new sfn.Map(this, "createUsersStep", {
      maxConcurrency: 5,
      itemsPath: "$.ids",
      resultPath: "$.users"
    }).iterator(createUserTask);

    const loadTesterFunction = new LambdaGoFunction(
      this,
      "loadTesterFunction",
      {
        name: "load-tester",
        environment: {
          USER_POOL_ID: userPool.userPoolId,
          USER_POOL_CLIENT_ID: userPoolClient.userPoolClientId,
          LOAD_TEST_ENDPOINT: api.apiEndpoint
        }
      }
    );
    loadTesterFunction.role?.addManagedPolicy(cognitoPolicy);
    loadTesterFunction.addToRolePolicy(
      new iam.PolicyStatement({
        effect: iam.Effect.ALLOW,
        resources: ["*"],
        actions: ["ssm:GetParameter", "ssm:DescribeParameter"]
      })
    );
    const loadTesterTask = new sfnTasks.LambdaInvoke(this, "loadTesterTask", {
      lambdaFunction: loadTesterFunction
    });
    const loadTestStep = new sfn.Map(this, "loadTestStep", {
      maxConcurrency: 5,
      itemsPath: "$.users"
    }).iterator(loadTesterTask);

    const definition = createUserIdsTask
      .next(createUsersStep)
      .next(loadTestStep);

    const machine = new sfn.StateMachine(this, "loadTestMachine", {
      definition: definition
    });

    new cdk.CfnOutput(this, "loadTestApiEndpoint", {
      value: api.apiEndpoint
    });
  }
}

interface LambdaGoFunctionProps extends Partial<GoFunctionProps> {
  name: string;
}
class LambdaGoFunction extends GoFunction {
  constructor(
    scope: cdk.Construct,
    id: string,
    { name, ...restOfProps }: LambdaGoFunctionProps
  ) {
    super(scope, id, {
      entry: join(__dirname, `../src/functions/${name}`),
      ...restOfProps
    });
  }
}
