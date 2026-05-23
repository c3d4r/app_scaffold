import { Construct } from 'constructs';
import * as cdk from 'aws-cdk-lib';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as acm from 'aws-cdk-lib/aws-certificatemanager';

export interface CloudFrontProps {
  staticBucket: s3.Bucket;
  generatedBucket: s3.Bucket;
  apiFnUrl: lambda.FunctionUrl;
  domainName?: string;
  certificate?: acm.ICertificate;
}

export class CloudFront extends Construct {
  public readonly distribution: cloudfront.Distribution;

  constructor(scope: Construct, id: string, props: CloudFrontProps) {
    super(scope, id);

    const fnUrlDomain = cdk.Fn.select(2, cdk.Fn.split('/', props.apiFnUrl.url));

    const apiOrigin = new origins.HttpOrigin(fnUrlDomain);

    const staticOrigin = origins.S3BucketOrigin.withOriginAccessControl(props.staticBucket);

    const generatedOrigin = origins.S3BucketOrigin.withOriginAccessControl(props.generatedBucket);

    if (props.domainName && props.certificate) {
      this.distribution = new cloudfront.Distribution(this, 'Distribution', {
        defaultBehavior: {
          origin: apiOrigin,
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
          allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
          cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
          originRequestPolicy: cloudfront.OriginRequestPolicy.ALL_VIEWER_EXCEPT_HOST_HEADER,
        },
        additionalBehaviors: {
          '/static/*': {
            origin: staticOrigin,
            viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
            cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
          },
          '/generated/*': {
            origin: generatedOrigin,
            viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
            cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
          },
        },
        priceClass: cloudfront.PriceClass.PRICE_CLASS_100,
        errorResponses: [
          {
            httpStatus: 404,
            responseHttpStatus: 200,
            responsePagePath: '/',
          },
        ],
        domainNames: [props.domainName],
        certificate: props.certificate,
      });
    } else {
      this.distribution = new cloudfront.Distribution(this, 'Distribution', {
        defaultBehavior: {
          origin: apiOrigin,
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
          allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
          cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
          originRequestPolicy: cloudfront.OriginRequestPolicy.ALL_VIEWER_EXCEPT_HOST_HEADER,
        },
        additionalBehaviors: {
          '/static/*': {
            origin: staticOrigin,
            viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
            cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
          },
          '/generated/*': {
            origin: generatedOrigin,
            viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
            cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
          },
        },
        priceClass: cloudfront.PriceClass.PRICE_CLASS_100,
        errorResponses: [
          {
            httpStatus: 404,
            responseHttpStatus: 200,
            responsePagePath: '/',
          },
        ],
      });
    }
  }
}
