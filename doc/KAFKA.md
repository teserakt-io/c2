
# Kafka notes

2 main Kafka distribution I've seen:
 - Apache Kafka: The Kafka service itself
 - Confluent plateform (https://www.confluent.io/product/confluent-platform/), sounds like Kafka enterprise edition, contains a kafka core and lots of tooling around

Golang client libs:
I've checked the following 2 and picked sarama which seems more up to date and maintained that the confluent one. Also I liked its API design more.
Also, confluent require additionnal system library to be installed on the host to work (librdkafka)
 - https://github.com/Shopify/sarama
 - https://github.com/confluentinc/confluent-kafka-go

Terminology:
 - https://www.tutorialspoint.com/apache_kafka/apache_kafka_fundamentals.htm
 - https://www.tutorialspoint.com/apache_kafka/apache_kafka_cluster_architecture.htm

Feature support per language:
This is the link I mentionned where it shows that golang clients are not supporting stream api
 - https://docs.confluent.io/current/clients/index.html#feature-support

Details of topics / partitions notion:
 - Kafka introduce a "partition" notion, which split a topic in smaller unit. From my understanding, their uses seems limited to load balancing between multiple consumers on the same topic (ie: round robin the message between each consumers) https://stackoverflow.com/questions/38024514/understanding-kafka-topics-and-partitions/51829144#51829144

Zookeeper:
 - Manage the "distribution" of Kafka service across multiple brokers. Transparent from the client side but still need to exists for Kafka to work.

Kafka & docker:
For a multi brokers in containers environment:
- https://hub.docker.com/r/wurstmeister/kafka
Confluent image was rather straightforward to spin up quickly a single broker:
- https://hub.docker.com/r/confluentinc/cp-kafka

Official guide:
 - https://kafka.apache.org/intro
 - https://cwiki.apache.org/confluence/display/KAFKA/A+Guide+To+The+Kafka+Protocol#AGuideToTheKafkaProtocol-Network
