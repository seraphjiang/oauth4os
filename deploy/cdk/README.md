# oauth4os CDK Deployment

Deploy oauth4os proxy + OpenSearch on AWS.

## Prerequisites

```bash
pip install aws-cdk-lib constructs
npm install -g aws-cdk
```

## Deploy

```bash
cd deploy/cdk
cdk deploy OAuth4OS \
  -c domain_base=huanji.profile.aws.dev \
  -c hosted_zone_id=Z07471832ZT9WEKD2RR39 \
  -c certificate_arn=arn:aws:acm:... \
  -c account=123456789012
```

## What it creates

- OpenSearch domain (t3.small, single-node, 20GB EBS)
- ECS Fargate service (256 CPU, 512MB) running oauth4os proxy
- ALB with HTTPS (custom domain + ACM cert)
- Route53 A record: `oauth4os.your-domain.com`
- VPC with 2 AZs + NAT gateway

## Cost

~$30/mo (OpenSearch t3.small + Fargate + NAT)

## Tear down

```bash
cdk destroy OAuth4OS
```
