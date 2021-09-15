## Kdelay
try to achive delay a message from kafka then emit it close to it expected to

### how it works
consuming from a kafka topic with header include the desired delay time.
*kdelay_release_at*
the value is either Unix or any GetTimeLayouts could figure out.
multiple cron jobs are fired up for scanning the messages needs to publish
after publish the message to the outgoing kafka topic, the message will be DELETED

### usage and configuration
- see confiugration under conf/.app.conf 
 rename conf/.app.conf to conf/app.conf to allow load credentials
 you will need give credeitials of 

#### incoming topic 
#### mysql 
#### outgoing topic   

- publish to the kdelay incoming topic with adding header key = kdelay_release_at
and the desired release time with a format in the list of GetTimeLayouts (in parse_time.go)



### design constraits

- time zone is convereted to local in database
- message is guaranteed NO SOONER than the desired kdelay_release_at
- if messages are overwhelmed the database performances, consider deploy multiple databases with same incoming and outgoing kafka topics
- message size is limited to 65,536 bytes
 

## backing services

- MySQL DB

- Redis

### Roadmap 
- accepting REST API or gRPC incoming message
- extend the message storage to s3 to break the limitation of mysql text type max size(65536bytes)
- extend to adopt RMQ