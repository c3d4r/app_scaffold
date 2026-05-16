import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';

import { StaticBucket } from './constructs/static-bucket';
import { GeneratedBucket } from './constructs/generated-bucket';
import { ApiLambda } from './constructs/api-lambda';
import { DurableLambda } from './constructs/durable-lambda';
import { CloudFront } from './constructs/cloudfront';

export interface AppScaffoldStackProps extends cdk.StackProps {
  bedrockModelId: string;
}

export class AppScaffoldStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: AppScaffoldStackProps) {
    super(scope, id, props);

    const staticAssets = new StaticBucket(this, 'StaticBucket');

    const generated = new GeneratedBucket(this, 'GeneratedBucket');

    const durable = new DurableLambda(this, 'DurableLambda', {
      generatedBucketName: generated.bucket.bucketName,
      bedrockModelId: props.bedrockModelId,
    });

    const api = new ApiLambda(this, 'ApiLambda', {
      generatedBucketName: generated.bucket.bucketName,
      durableFunctionName: durable.fn.functionName,
    });

    const cf = new CloudFront(this, 'CloudFront', {
      staticBucket: staticAssets.bucket,
      generatedBucket: generated.bucket,
      apiFnUrl: api.functionUrl,
    });

    new s3deploy.BucketDeployment(this, 'DeployStaticAssets', {
      sources: [s3deploy.Source.asset('../dist/static')],
      destinationBucket: staticAssets.bucket,
      distribution: cf.distribution,
      distributionPaths: ['/static/*'],
    });

    new cdk.CfnOutput(this, 'CloudFrontDomain', {
      value: cf.distribution.domainName,
      description: 'CloudFront distribution domain',
    });
  }
}
