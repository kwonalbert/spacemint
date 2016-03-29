# Spacemint

This is a prototype of Spacemint (Spacecoin), which is described in
more detail at
https://eprint.iacr.org/2015/528.pdf (Cryptocurrency)
https://eprint.iacr.org/2013/796.pdf (Proof-of-space; PoS).
This README explains some implementation details.

###The proof-of-space part of this is being re-implemented,
and a version of it is already moved to https://github.com/kwonalbert/pospace.

##Structure of the graph

For proof-of-space, the graphs are generated once, and are read only
for rest of its life-time. Moreover, the only operations we need to do
are parent look up and hash lookups. We therefore pick a simple graph
representation in the file system, and avoid using a more
sophisticated solution like graph database.

The graphs used in the prototype from PTC76 (Paul, Tarjan and Celoni).
The graph is recursively generated. Each node is just a hash, and you
can directly access the node by reading hash at id * size of hash.
The code is written in a way to minimize the HDD head movement, at the
cost of some minor compute during initialization.

##Directory Structure
block/          Cryptocurrency block files

pos/            Proof-of-Space implementation

util/           Various utilities used by block and pos