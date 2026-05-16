#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { AppScaffoldStack } from '../lib/app-scaffold-stack';

const app = new cdk.App();

new AppScaffoldStack(app, 'AppScaffoldStack', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION,
  },
  bedrockModelId: process.env.BEDROCK_MODEL_ID ?? 'us.anthropic.claude-3-5-sonnet-20241022-v2:0',
});
