#!/usr/bin/env python3
import os
import aws_cdk as cdk
from stack import OAuth4OSStack

app = cdk.App()
account = app.node.try_get_context("account") or os.environ.get("CDK_DEFAULT_ACCOUNT", "")
region = app.node.try_get_context("region") or os.environ.get("CDK_DEFAULT_REGION", "us-west-2")

OAuth4OSStack(app, "OAuth4OS", env=cdk.Environment(account=account, region=region))
app.synth()
