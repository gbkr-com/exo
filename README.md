# exo

> from Greek, exō ‘outside’.

## Purpose

`exo` is an exoskeleton for automated execution, written in Go. It is intended to demonstrate some key concepts in designing a process to manage many simultaneous executions. It does not show how to execute, let alone execute in a safe and efficient manner: that is another story.

`exo` is written in Go for a number of reasons. One is the natural way Go supports concurrency. Another is, having done this in real operations, it is more than capable of supporting real time trading and, at the same time, the language and compilation speed make it easy for someone with a Python background to work on the code.

## Concepts

### Operational

Real time execution needs real time data. In many such cases the arrival rate of market data, even Level I (quote) data, can outstrip the ability to send and persist orders for the counterparty. One solution is 'co-location' and the dedication to handle _every_ tick. Another, more economic, approach is 'conflation'.

> the merging of two or more sets of information ... into one

`exo` uses conflation in two ways. One, `exo` fans-out ticker data from a single subscriber to the subscribing delegates, via a dispatcher. When that dispatcher is busy, ticker data is conflated until it can be presented for dispatch. That is conflation of the same data type.

The second way is when quote, trade, instruction and fill information is to be presented to a delegate. One approach is to treat all these independently. However, delegates can then become frantic in making decisions they regret because they did not have a full picture before each decision. `exo` calms this behaviour by conflating all these updates into one package for decision making.

In keeping with standard practice, the side and symbol of an order are immutable after creation. With the order ID assigned by `exo`, they form the fundamental identity of an order.

### Technical

#### Components

See:
- [Dispatcher](run/dispatcher.go)
- [OrderProcess](run/order-process.go)
- [Delegate](run/delegate.go)

The central component is `Dispatcher`. When it receives a new order, it creates a goroutine for an `OrderProcess` to work on that order. An `OrderProcess` is simply a link to a `Delegate` which does the real work on the order.

The `Dispatcher` and `OrderProcess` are the 'container' surrounding a delegate. Both do not need to know much about an order apart from its identity. Both use Go generics so that the basic `mkt.Order` can be extended without affecting how `Dispatcher` and `OrderProcess` work.

#### Channels

Go channels are a natural way to make the dispatcher code wholly event driven through the `select` statement. However, channels have capacity and will block when full. `exo` uses the `utl.ConflatingQueue` type which presents a channel that can be used in a `select` yet, until the queue is popped, data is still being conflated and not lost.

The number of steps from data arrival to a delegate is minimal. For market data it is one conflating queue to the `Dispatcher`, then one more to the `Delegate`.

#### Persistence, logging ...

`exo` does not prescribe any of these. A `Delegate` is free to make those choices as it is `outside` of the container code.