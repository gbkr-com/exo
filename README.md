# exo

> from Greek, exō ‘outside’.

## Purpose

`exo` is an exoskeleton for automated execution, written in Go. It is intended to demonstrate some key concepts in designing a process to manage many simultaneous executions. It does not show how to execute, let alone execute in a safe and efficient manner: that is another conversation.

`exo` is written in Go for a number of reasons. One is the natural way Go supports concurrency. Another is, having done this IRL, Go is more than capable of supporting real time trading and, at the same time, the language and compilation speed make it easy for someone with a Python background to work on the code.

## Concepts

### Operational

Real time execution needs real time data. In many such cases the arrival rate of market data, even Level I (quote) data, can outstrip the speed of sending orders to the counterparty. One solution is 'co-location' and the dedication to handle ***every*** tick. Another, more economic, approach is 'conflation'.

> the merging of two or more sets of information ... into one

`exo` uses conflation in two ways. One, `exo` fans-out ticker data from a single subscriber to the subscribing delegates, via a dispatcher. When that dispatcher is busy, ticker data is conflated until it can be presented for dispatch. That is conflation of the same data type.

The second way is when quote, trade, instruction and fill information is to be presented to a delegate. One approach is to treat all these independently. However, delegates can then become frantic, making decisions they regret because they did not have a full picture before each decision. `exo` calms this behaviour by conflating all these updates into one struct for decision making.

In keeping with standard practice, the side and symbol of an order are immutable after creation. With the order ID assigned by `exo`, they form the fundamental identity of an order.

### Technical

#### Components

See:
- [Dispatcher](run/dispatcher.go)
- [OrderProcess](run/order-process.go)
- [Delegate](run/delegate.go)

The `Dispatcher` receives orders and market data. For each new order it creates an `OrderProcess`, and launches a goroutine to manage work on that order. The `OrderProcess` has no order handling logic: it passes all updates to its associated `Delegate` to do that work.

The `Dispatcher` and `OrderProcess` are the 'container' surrounding a delegate. Both do not need to know much about an order apart from its identity. Both use Go generics so that the basic `mkt.Order` can be extended without affecting how `Dispatcher` and `OrderProcess` work.

#### Channels

Go channels are a natural way to make the dispatcher code wholly event driven through the `select` statement. However, channels have capacity and will block when full. `exo` uses the `utl.ConflatingQueue` type which presents a channel that can be used in a `select` yet, until the queue is popped, data is still being conflated and not lost.

This conflation pattern was working for equity markets in 2012. Then it was written in Java. It is now much simpler, and more efficient, in Go.

#### Persistence, logging ...

`exo` does not prescribe any of these. A `Delegate` is free to make those choices as it is 'outside' of the container code.

## Performance

I/O dominates compute. And, for many crypto markets, transaction I/O dominates market data I/O. Rate limits can dominate everything.

### Compute

The number of steps from data arrival to a delegate is minimal. For market data it is one conflating queue to the `Dispatcher`, then one more to the `Delegate`.

[BenchmarkOrderProcess](run/order-process_test.go) benchmarks a quote from entry to the `OrderProcess` to receipt by the `Delegate`, where it is forced to acknowledge the quote before the benchmark continues, thereby disabling conlfation. On an Apple M1, that benchmarks at ~430ns, or ~2m operations per second.

### Cloud

Looking at AWS latency with [CloudPing](https://www.cloudping.co/grid), with a 'co-lo' strategy the minimum latency is single digit milliseconds, for example around 1-2 milliseconds for Tokyo (Binance). This is equivalent to 500 messages per second.

## Example

See [Example](cmd/example/main.go)

The example illustrates extending the order object and adds a rudimentary execution algorithm. It requires Redis. Run the program with:
```
make build
make run-example
```
Enter the order details via Postman or similar, then something like this should appear on the terminal:
```
20:58:55 exo % make run-example
trade 0.42582859 @ 59988.41
trade 0.00442362 @ 59988.41
trade 0.06974779 @ 59988.61
done
```
