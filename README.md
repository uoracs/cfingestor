# cfingestor

Simple web server that accepts a specifically-formatted payload and outputs Coldfront-compatible import files.

The input data should look like this:

```json
{
  "users": [
    {
        "username":  "username1",
        "firstname":  "userfirstname",
        "lastname":  "userlastname"
    },
    {
        "username":  "user2",
        "firstname":  "user2firstname",
        "lastname":  "user2lastname"
    }
  ],
  "projects": [
    {
      "project": "project1",
      "owner": "username1",
      "admins": [
        "username1"
      ],
      "users": [
        "username1",
      ]
    },
    {
      "project":  "project2",
      "owner":  "user2",
      "admins":  [
        "user2"
      ],
      "users":  [
        "user2",
        "username1"
      ]
    },
  ]
}
```
