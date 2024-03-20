# cfingestor

Simple web server that accepts a specifically-formatted payload and outputs Coldfront-compatible import files.

To run it, activate the coldfront virtual environment, then run the server like this:

```bash
coldfront shell < /path/to/main.py
```

Yes this is super ugly but it's just a stop-gap until we can convert to coldfront full-time.

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

