resource "aws_iam_role" "lambda" {
  name = "${var.function_name}-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  role       = aws_iam_role.lambda.name
}

resource "aws_iam_role_policy" "lambda_dynamodb" {
  name = "${var.function_name}-dynamodb-${var.environment}"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = [
          var.dynamodb_table_arn,
          "${var.dynamodb_table_arn}/index/*"
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_cognito" {
  name = "${var.function_name}-cognito-${var.environment}"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cognito-idp:AdminCreateUser",
          "cognito-idp:AdminGetUser",
          "cognito-idp:AdminInitiateAuth",
          "cognito-idp:AdminRespondToAuthChallenge",
          "cognito-idp:AdminConfirmSignUp"
        ]
        Resource = var.cognito_pool_arn
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_ses" {
  name = "${var.function_name}-ses-${var.environment}"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ses:SendEmail",
          "ses:SendRawEmail",
          "sesv2:SendEmail"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_sqs" {
  count = length(var.sqs_queue_arns) > 0 ? 1 : 0

  name = "${var.function_name}-sqs-${var.environment}"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = var.sqs_queue_arns
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_sqs_consume" {
  count = length(var.sqs_consume_arns) > 0 ? 1 : 0

  name = "${var.function_name}-sqs-consume-${var.environment}"
  role = aws_iam_role.lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes"
        ]
        Resource = var.sqs_consume_arns
      }
    ]
  })
}

resource "aws_lambda_function" "main" {
  function_name = "${var.function_name}-${var.environment}"
  role          = aws_iam_role.lambda.arn
  handler       = var.handler
  runtime       = var.runtime
  filename      = var.source_file
  memory_size   = var.environment == "prod" ? 512 : 256
  timeout       = var.lambda_timeout

  source_code_hash = filebase64sha256(var.source_file)

  environment {
    variables = var.environment_variables
  }

  depends_on = [
    aws_iam_role_policy_attachment.lambda_basic,
  ]
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${aws_lambda_function.main.function_name}"
  retention_in_days = var.environment == "prod" ? 7 : 3 # Prod: 7 dias, Dev: 3 dias
}