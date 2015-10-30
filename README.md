# Spacemint

This is a prototype of Spacemint (Spacecoin), which is described in
more detail at
https://eprint.iacr.org/2015/528.pdf (Cryptocurrency)
https://eprint.iacr.org/2013/796.pdf (Proof-of-space; PoS).
This README explains some implementation details.

##Structure of the graph

For proof-of-space, the graphs are generated once, and are read only
for rest of its life-time. Moreover, the only operations we need to do
are parent look up and hash lookups. We therefore pick a simple graph
representation in the file system, and avoid using a more
sophisticated solution like graph database.

The graphs used in the prototype from PTC76 (Paul, Tarjan and Celoni).
The graph is recursively generated. Each node is a json encoded file,
that consits of 1. id, 2. hash, 3. list of parents. The parents are
the file names that hold the parents for this node.