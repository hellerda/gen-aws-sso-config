# Overview

This program will automatically generate an ```.aws/config``` file for a user, based on the accounts and roles available to the user upon SSO login.  The config can be used for SSO login through Identity Center, for AWS CLI or other tools that read the ```.aws/config```.

The program will open a browser window for you to SSO login.  Afterward, close the tab and hit return.  The program will then build your config and output it to stdout.  To use, add the snip to your ```.aws/config``` file.

Based on (https://github.com/aws/aws-sdk-go-v2/issues/1222).

# Build
```
$ go build
```

# Usage
```
$ gen-aws-sso-config --start-url "https://d-987654321d.awsapps.com/start" -sso-session-name "my-sso" -sso-region "us-east-1"
```

# Notes

- Although you provide the ```sso-region```, the region of the Identity Center instance, the program does not know what region you want to use for each profile entry.  By default it adds a region pointing the same as sso-region, but this entry is commented out.  If you want to the set region for a profile, edit the file accordingly.
