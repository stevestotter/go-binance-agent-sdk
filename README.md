# Go Binance Agent SDK

Using [Binance](https://www.binance.com/en) feeds of cryptocurrency markets to instruct algorithmic traders (agents).

The SDK aims to:
- Provide the community with a framework in order to build their own agents and take advantage of algorithmic trading
- Provide common interfaces for agents, feeds and exchange interactions
- Provide an [example agent](examples/simple-agent) & exchange simulator to be able to get started quickly and run experiments during development


**Please Note:** This is still at a very early stage so code structure, interfaces and even the name of this repository are likely to change.


## Makefile helpers

``` make deps``` - grabs dependencies needed for the project

``` make mocks ``` - cleans and re-generates mocks for testing

``` make test``` - runs the tests with race detection