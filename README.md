# Load testing APIGW with AWS Step Functions

Inspired by [this post on AWS Compute blog](https://aws.amazon.com/blogs/compute/using-serverless-to-load-test-amazon-api-gateway-with-authorization/)

## Learnings

- Writing a middleware for the `go-lambda` is not that hard

- _SecretsManager_ secrets have to be _scheduled_ for deletion, you cannot delete them instantly

- The validation error message for _SSM_ `putParameter` can be misleading. I've encountered messages like "parameters name cannot start with `ssm` prefix". Of course the name I was specifying DID NOT start with `ssm`.

- To be able to _signUp_ user as an admin, I still had to specify `selfSignUpEnabled: true`. Giving it a bit more thought, it makes sense.
