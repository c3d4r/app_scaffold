import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';
import * as acm from 'aws-cdk-lib/aws-certificatemanager';
import * as route53 from 'aws-cdk-lib/aws-route53';
import * as route53targets from 'aws-cdk-lib/aws-route53-targets';

import { StaticBucket } from './constructs/static-bucket';
import { GeneratedBucket } from './constructs/generated-bucket';
import { ApiLambda } from './constructs/api-lambda';
import { DurableLambda } from './constructs/durable-lambda';
import { CloudFront } from './constructs/cloudfront';
import { CognitoAuth } from './constructs/cognito';

export interface AppScaffoldStackProps extends cdk.StackProps {
  bedrockModelId: string;
}

const DOMAIN_NAME = 'app.ced4r.link';
const HOSTED_ZONE_ID = 'Z0428298AG7J0BEZHW8U';

export class AppScaffoldStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: AppScaffoldStackProps) {
    super(scope, id, props);

    const hostedZone = route53.HostedZone.fromHostedZoneAttributes(this, 'HostedZone', {
      hostedZoneId: HOSTED_ZONE_ID,
      zoneName: 'ced4r.link',
    });

    const certificate = new acm.Certificate(this, 'Certificate', {
      domainName: DOMAIN_NAME,
      validation: acm.CertificateValidation.fromDns(hostedZone),
    });

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
      domainName: DOMAIN_NAME,
      certificate,
    });

    new route53.ARecord(this, 'ARecord', {
      zone: hostedZone,
      recordName: 'app',
      target: route53.RecordTarget.fromAlias(
        new route53targets.CloudFrontTarget(cf.distribution)
      ),
    });

    new route53.AaaaRecord(this, 'AaaaRecord', {
      zone: hostedZone,
      recordName: 'app',
      target: route53.RecordTarget.fromAlias(
        new route53targets.CloudFrontTarget(cf.distribution)
      ),
    });

    const callbackUrl = `https://${DOMAIN_NAME}/auth/callback`;
    const logoutUrl = `https://${DOMAIN_NAME}/about`;

    const cognito = new CognitoAuth(this, 'CognitoAuth', {
      callbackUrl,
      logoutUrl,
    });

    api.addCognitoEnv({
      userPoolId: cognito.userPool.userPoolId,
      clientId: cognito.userPoolClient.userPoolClientId,
      clientSecret: cognito.userPoolClient.userPoolClientSecret,
      domain: `app-scaffold-${this.account}`,
      region: this.region,
      callbackUrl,
    });

    new s3deploy.BucketDeployment(this, 'DeployStaticAssets', {
      sources: [s3deploy.Source.asset('../dist/static')],
      destinationBucket: staticAssets.bucket,
      destinationKeyPrefix: 'static/',
      distribution: cf.distribution,
      distributionPaths: ['/static/*'],
    });

    new cdk.CfnOutput(this, 'CloudFrontDomain', {
      value: DOMAIN_NAME,
      description: 'CloudFront custom domain',
    });

    new cdk.CfnOutput(this, 'CognitoDomain', {
      value: cognito.domainUrl,
      description: 'Cognito hosted UI domain',
    });
  }
}
