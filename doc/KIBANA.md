# Kibana

Web interface: [http://localhost:5601](http://localhost:5601)

Preconfigured dashboards are provided in the [kibana_objects.json](../configs/kibana_objects.json) file.
It can be imported by browsing to [http://localhost:5601/app/kibana#/management/kibana/objects](http://localhost:5601/app/kibana#/management/kibana/objects) and use the import function to load the file.

## Dashboards

Once the preconfigured objects imported, dashboards are accessible at [http://localhost:5601/app/kibana#/dashboards](http://localhost:5601/app/kibana#/dashboards)

### C2

This dashboard provide visualition over the C2 componements such as the GRPC/HTTP server, the E4 service or the MQTT client.
It can be used to see the amount and kind of requests hitting the C2.


### C2 - Errors

This dashboard will display any errors collected in the application logs, as well as a graphic of errors per protocols over time for an easier readability.

### C2 - Messages

This dashboard collect metrics of the message received by the C2 on the monitored topics.
It displays the message logs, and rate over time in both numeric and graphical representation.


## Common issues

### Empty dashboards

#### Check time frame configuration

Kibana rely on a global time frame configuration to know what it should display, which is persisted accross all its screens.
It can be found on the top right of the screen and provide the ability to specify a time range from dates, and some shortcuts like "Last 1 hour".
Make sure that the time frame is set properly for the data you're trying to visualize.

#### Check filters

When the time frame is correct but the dashboard still refuse to display the data, it can be because some *filters* are on. The filter bar is located just under the search bar, on top of the screen. They allow to restrict all the visualisations and queries to match the defined filters. And they can be set as simply as by clicking on a label or in a graph. The must be manually removed (hover of the filter > trashcan icon) or toggle off (hover on the filter > checkbox icon) to restore the original queries.

### Dashboard not displaying new data / not auto refreshing

Kibana auto refreshing works in a same way as the time frame configuration and is persisted across screens. It's immediatly on the left of the time frame configuration, on the top right of the screen. Make sure to turn it on to see it display new logs as they keep arriving.
