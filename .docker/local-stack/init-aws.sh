awslocal iam create-user --user-name auto-roto
awslocal iam create-access-key --user-name auto-roto > /test-tmp/access-key-secret-auto-roto.json
awslocal iam put-user-policy --user-name auto-roto --policy-name YourPolicyName --policy-document file:///auto-roto-policy.json
