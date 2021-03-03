from .common import get_admin_client, LINODE_ACCESSKEY, AWS_SECRET_ACCESS_KEY, \
    AWS_ACCESS_KEY_ID
import os
#RANCHER_CLUSTER NAME, CATTLE_TEST_URL, USER_TOKEN
credentials={"linode" : None, "digital_ocean" : None}
DIGITAL_OCEAN_ACCESSKEY = os.environ.get('RANCHER_DIGITAL_OCEAN_ACCESSKEY', "None")
def test_create_linode_credential():
    client = get_admin_client()
    linode_cloud_credential_config = {"token": LINODE_ACCESSKEY}
    linode_cloud_credential = client.create_cloud_credential(
        linodecredentialConfig=linode_cloud_credential_config,
        name="automated-linode-credentials"
    )
    
def test_create_digital_ocean_credential():
    client = get_admin_client()
    digital_ocean_cloud_credential_config = {"accessToken": DIGITAL_OCEAN_ACCESSKEY}
    digital_ocean_cloud_credential = client.create_cloud_credential(
        digitaloceancredentialConfig=digital_ocean_cloud_credential_config,
        name="automated-digital-ocean-credentials"
    )

def test_create_aws_credential():
    client = get_admin_client()
    aws_cloud_credential_config = {
        "accessKey": AWS_ACCESS_KEY_ID,
        "secretKey": AWS_SECRET_ACCESS_KEY}
    aws_cloud_credential = client.create_cloud_credential(
        amazonec2credentialConfig=aws_cloud_credential_config,
        name="automated-aws-credentials"
    )