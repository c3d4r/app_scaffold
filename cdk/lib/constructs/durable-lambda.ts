import { Construct } from 'constructs';
import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as iam from 'aws-cdk-lib/aws-iam';

export interface DurableLambdaProps {
  generatedBucketName: string;
  bedrockModelId: string;
}

export class DurableLambda extends Construct {
  public readonly fn: lambda.Function;

  constructor(scope: Construct, id: string, props: DurableLambdaProps) {
    super(scope, id);

    this.fn = new lambda.Function(this, 'Function', {
      runtime: lambda.Runtime.PYTHON_3_13,
      architecture: lambda.Architecture.ARM_64,
      handler: 'main.handler',
      code: lambda.Code.fromAsset('../dist/durable'),
      memorySize: 256,
      timeout: cdk.Duration.seconds(30),
      environment: {
        GENERATED_BUCKET: props.generatedBucketName,
        BEDROCK_MODEL_ID: props.bedrockModelId,
      },
      loggingFormat: lambda.LoggingFormat.JSON,
      applicationLogLevelV2: lambda.ApplicationLogLevel.INFO,
    });

    this.fn.addToRolePolicy(
      new iam.PolicyStatement({
        actions: ['s3:GetObject', 's3:PutObject'],
        resources: [
          `arn:aws:s3:::${props.generatedBucketName}`,
          `arn:aws:s3:::${props.generatedBucketName}/*`,
        ],
      })
    );

    this.fn.addToRolePolicy(
      new iam.PolicyStatement({
        actions: ['bedrock:InvokeModel', 'bedrock:InvokeModelWithResponseStream'],
        resources: ['*'],
      })
    );
  }
}
