import { Construct } from 'constructs';
import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as iam from 'aws-cdk-lib/aws-iam';

export interface ApiLambdaProps {
  generatedBucketName: string;
  durableFunctionName: string;
}

export class ApiLambda extends Construct {
  public readonly fn: lambda.Function;
  public readonly functionUrl: lambda.FunctionUrl;

  constructor(scope: Construct, id: string, props: ApiLambdaProps) {
    super(scope, id);

    this.fn = new lambda.Function(this, 'Function', {
      runtime: lambda.Runtime.PROVIDED_AL2023,
      architecture: lambda.Architecture.ARM_64,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../dist/api'),
      memorySize: 256,
      timeout: cdk.Duration.seconds(10),
      environment: {
        APP_ENV: 'production',
        GENERATED_BUCKET: props.generatedBucketName,
        DURABLE_LAMBDA_NAME: props.durableFunctionName,
      },
      loggingFormat: lambda.LoggingFormat.JSON,
      applicationLogLevelV2: lambda.ApplicationLogLevel.INFO,
    });

    this.fn.addToRolePolicy(
      new iam.PolicyStatement({
        actions: ['s3:GetObject', 's3:PutObject'],
        resources: [`arn:aws:s3:::${props.generatedBucketName}/*`],
      })
    );

    this.fn.addToRolePolicy(
      new iam.PolicyStatement({
        actions: ['s3:ListBucket'],
        resources: [`arn:aws:s3:::${props.generatedBucketName}`],
      })
    );

    this.fn.addToRolePolicy(
      new iam.PolicyStatement({
        actions: ['lambda:InvokeFunction'],
        resources: ['*'],
        conditions: {
          StringEquals: {
            'lambda:FunctionName': props.durableFunctionName,
          },
        },
      })
    );

    this.functionUrl = this.fn.addFunctionUrl({
      authType: lambda.FunctionUrlAuthType.NONE,
    });
  }
}
