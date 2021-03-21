# Go Binance Simple Agent - Example

A very simple agent, showing how to use the Go Binance Agent SDK for algorithmic trading.


## Analysis with Kibana

By using Docker Compose, this agent will be brought up with a full ELK stack. Once up, you can navigate to Kibana at the following web address:

http://localhost:5601/

By creating dashboards on trades and market activity, you can keep track of how the agent performs.

## Makefile helpers

``` make deps``` - grabs dependencies needed for the agent

``` make run-simple-agent ``` - runs the agent locally

``` make docker-stack-up``` - brings up the agent and a whole ELK stack in containers, in detached mode

``` make docker-stack-down``` - brings down all containers

``` make docker-stack-destroy``` - brings down all containers and destroys volumes