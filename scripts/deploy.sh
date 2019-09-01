#!/bin/bash
#define environment variables
server_deploy_file_path="/workspace/docker-stack/goblog/cd.sh"

echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
if [ $? -neq 0 ]; then
    return 1
fi
docker push $IMAGE_TAG
openssl aes-256-cbc -k "$super_secret_password" -in super_secret.txt.enc -out super_secret.txt -d
chmod 400 super_secret.txt
ssh -o StrictHostKeyChecking=no -i super_secret.txt root@47.92.105.16 $server_deploy_file_path $TRAVIS_COMMIT
if [ $? -ne 0 ]; then
    exit 1
else
    echo "-----------------------------"
    echo "deploy completed successfully"
    echo "-----------------------------"
fi
