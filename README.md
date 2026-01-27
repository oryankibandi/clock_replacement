# CLOCK Replacement Algorithm - In Progress

This is an implementation of the CLOCK replacement algorithm popularly used in storage engines and databases such as PostgreSQL.

It is known for it's simplicity and high concurrency compared to LRU, an alternative cache eviction policy.

### How it works

CLOCK replacement algorithm utilizes a circular buffer, arranged as a linked list,  with each item containing a reference bit and an access bit, and a clock hand that points to a a node in the linked list.

[circular_buffer](/docs//images/circular_buffer.png)

Each time a new item needs to be added and the cache is full, the clock hand checks the item it currently points to.

If the access bit is set, it proceeds to the next item. If access bit is not set, but reference bit is set, we unset the reference bit and proceed to the next item. This gives an item a "second chance". If both the access bit and reference bit are unset, the item hasn't been recently used and is evicted and the new item is added.

### Drawbacks

CLOCK replacement algorithm is efficient but has  a relatively low hit-rate. This has given rise to optimizations to the CLOCK replacement algorithm like RockDB's **HyperClockCache** and **[Clock-Pro](https://www.usenix.org/legacy/event/usenix05/tech/general/full_papers/jiang/jiang.pdf)**, used in **[Pebble](https://github.com/cockroachdb/pebble/)** - CockroachDB's storage engine.
