
## Features

### Dry run capability

```bash
go run main.go -dry-run

# in the comand line output
Will delete namespace user1
Will delete pv user1-dsmlp-datasets
```

## Configurations

Configurations:
- awsed API key [x] - put into a .env var as of right now, will be moved to .json in a future
- awsed API URL [x]
- list of persistent volumes to delete, ex. {user}-dsmlp-datasets, {user}-dsmlp-datasets-nfs

## List of Persistent Volumes

```
<user>-dsmlp-datasets                   5Gi        ROX            Retain           Bound         <user>/dsmlp-datasets                                               144d
<user>-dsmlp-datasets-nfs               5Gi        ROX            Retain           Bound         <user>/dsmlp-datasets-nfs                                           144d
<user>-home                             5Gi        RWX            Retain           Bound         <user>/home                                                         144d
<user>-home-nfs                         5Gi        RWX            Retain           Bound         <user>/home-nfs                                                     144d
<user>-nbgrader                         5Gi        ROX            Retain           Bound         <user>/nbgrader                                                     144d
<user>-support                          5Gi        ROX            Retain           Bound         <user>/support                                                      144d
<user>-teams
```