"""CDK stack for oauth4os demo — proxy + OpenSearch on AWS.

Deploys:
- OpenSearch domain (single-node, t3.small)
- ECS Fargate service running oauth4os proxy
- ALB with custom domain (oauth4os.your-domain.com)
- Route53 A record
"""
import os

from aws_cdk import (
    Stack, Duration, RemovalPolicy, CfnOutput,
    aws_ec2 as ec2,
    aws_ecs as ecs,
    aws_ecs_patterns as ecs_patterns,
    aws_opensearchservice as opensearch,
    aws_route53 as route53,
    aws_route53_targets as r53_targets,
    aws_certificatemanager as acm,
    aws_iam as iam,
)
from constructs import Construct


class OAuth4OSStack(Stack):
    def __init__(self, scope: Construct, id: str, **kwargs):
        super().__init__(scope, id, **kwargs)

        domain_base = self.node.try_get_context("domain_base") or ""
        hosted_zone_id = self.node.try_get_context("hosted_zone_id") or ""
        cert_arn = self.node.try_get_context("certificate_arn") or ""

        # VPC
        vpc = ec2.Vpc(self, "Vpc", max_azs=2, nat_gateways=1)

        # OpenSearch domain (single-node demo)
        os_domain = opensearch.Domain(self, "OpenSearch",
            version=opensearch.EngineVersion.OPENSEARCH_2_17,
            capacity=opensearch.CapacityConfig(
                data_node_instance_type="t3.small.search",
                data_nodes=1,
            ),
            ebs=opensearch.EbsOptions(volume_size=20),
            vpc=vpc,
            vpc_subnets=[ec2.SubnetSelection(subnet_type=ec2.SubnetType.PRIVATE_WITH_EGRESS)],
            removal_policy=RemovalPolicy.DESTROY,
            fine_grained_access_control=opensearch.AdvancedSecurityOptions(
                master_user_name="admin",
                master_user_password=self.node.try_get_context("os_admin_password") or None,
            ),
            node_to_node_encryption=True,
            encryption_at_rest=opensearch.EncryptionAtRestOptions(enabled=True),
            enforce_https=True,
        )

        # ECS cluster
        cluster = ecs.Cluster(self, "Cluster", vpc=vpc)

        # Proxy image from local Dockerfile
        proxy_image = ecs.ContainerImage.from_asset(
            os.path.join(os.path.dirname(__file__), "..", ".."),
        )

        # ALB + Fargate service
        proxy_domain = f"oauth4os.{domain_base}" if domain_base else ""

        svc_props = dict(
            cluster=cluster,
            cpu=256,
            memory_limit_mib=512,
            desired_count=1,
            task_image_options=ecs_patterns.ApplicationLoadBalancedTaskImageOptions(
                image=proxy_image,
                container_port=8443,
                environment={
                    "OAUTH4OS_UPSTREAM_ENGINE": f"https://{os_domain.domain_endpoint}",
                    "OAUTH4OS_UPSTREAM_DASHBOARDS": f"https://{os_domain.domain_endpoint}/_dashboards",
                    "OAUTH4OS_LISTEN": ":8443",
                },
            ),
            public_load_balancer=True,
        )

        # Add custom domain + cert if configured
        if cert_arn and proxy_domain:
            zone = route53.HostedZone.from_hosted_zone_attributes(self, "Zone",
                hosted_zone_id=hosted_zone_id,
                zone_name=domain_base,
            )
            cert = acm.Certificate.from_certificate_arn(self, "Cert", cert_arn)
            svc_props["domain_name"] = proxy_domain
            svc_props["domain_zone"] = zone
            svc_props["certificate"] = cert

        service = ecs_patterns.ApplicationLoadBalancedFargateService(self, "Proxy", **svc_props)

        # Allow proxy to reach OpenSearch
        os_domain.connections.allow_from(service.service, ec2.Port.tcp(443))

        # Health check
        service.target_group.configure_health_check(
            path="/health",
            healthy_http_codes="200",
            interval=Duration.seconds(30),
        )

        CfnOutput(self, "ProxyUrl", value=service.load_balancer.load_balancer_dns_name)
        CfnOutput(self, "OpenSearchEndpoint", value=os_domain.domain_endpoint)
        if proxy_domain:
            CfnOutput(self, "DemoUrl", value=f"https://{proxy_domain}")
