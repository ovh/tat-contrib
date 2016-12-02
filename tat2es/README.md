[![GoDoc](https://godoc.org/github.com/ovh/tat-contrib/tat2es?status.svg)](https://godoc.org/github.com/ovh/tat-contrib/tat2es)
[![Go Report Card](https://goreportcard.com/badge/ovh/tat-contrib/tat2es)](https://goreportcard.com/report/ovh/tat-contrib/tat2es)

# Tat2ES

<img align="right" src="https://raw.githubusercontent.com/ovh/tat/master/tat.png">

Documentation: https://ovh.github.io/tat/ecosystem/tat2es/

# ElasticSearch Configuration

- Create an index "yourIndex"
- For this index, create a mapping with this curl:

```bash
curl -XPUT https://yourHost:9200/yourIndex/tatmessage/_mapping/ -d '{
  "tatmessage": {
    "dynamic_templates": [
      {
        "notanalyzed": {
          "match": "*",
          "match_mapping_type": "string",
          "mapping": {
            "type": "string",
            "index": "not_analyzed"
          }
        }
      }
    ],
    "properties": {
      "Author": {
        "properties": {
          "fullname": {
            "type": "string",
            "fields": {
              "raw": {
                "index": "not_analyzed",
                "type": "string"
              }
            }
          },
          "username": {
            "type": "string",
            "fields": {
              "raw": {
                "index": "not_analyzed",
                "type": "string"
              }
            }
          }
        }
      },
      "DateCreation": {
        "type": "date",
        "format": "strict_date_optional_time||epoch_millis"
      },
      "DateUpdate": {
        "type": "date",
        "format": "strict_date_optional_time||epoch_millis"
      },
      "Delta": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      },
      "ID": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      },
      "InReplyOfID": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      },
      "Labels": {
        "properties": {
          "color": {
            "type": "string"
          },
          "text": {
            "type": "string",
            "fields": {
              "raw": {
                "index": "not_analyzed",
                "type": "string"
              }
            }
          }
        }
      },
      "NbLikes": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      },
      "Tags": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      },
      "Text": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      },
      "Topic": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      },
      "Urls": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      },
      "UserMentions": {
        "type": "string",
        "fields": {
          "raw": {
            "index": "not_analyzed",
            "type": "string"
          }
        }
      }
    }
  }
}'
```
# Hacking

```bash
mkdir -p $GOPATH/src/github.com/ovh
cd $GOPATH/src/github.com/ovh
git clone git@github.com:ovh/tat-contrib.git
cd tat-contrib/tat2es/api
go build
./api -h
```

You've developed a new cool feature? Fixed an annoying bug? We'd be happy
to hear from you! Make sure to read [CONTRIBUTING.md](./CONTRIBUTING.md) before.

# License

This work is under the BSD license, see the [LICENSE](LICENSE) file for details.
