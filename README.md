## Spacecoin

This is a prototype of Spacecoin, which is described in more detail at
https://eprint.iacr.org/2015/528.pdf (Cryptocurrency)
and
https://eprint.iacr.org/2013/796.pdf (Proof-of-space; PoS).
This README explains some implementation details.

#Structure of the graph

For proof-of-space, the graphs are generated once, and are read only
for rest of its life-time. Moreover, the only operations we need to do
are parent look up. We therefore pick a simple graph representation
in the file system, and avoid using a more sophisticated solution like
graph database.

There are N vertices ({0, ..., N-1}). Each vertex will be a directory
that contains several files. The directory name will be the vertex
ID. First, it contains a file called 'hash' which is an evaluation of
H described in the PoS paper. Vertex also contains symlinks to it's
parents, one symlink per parent. This makes find parents easy to
handle. Note that this replaces edges in the traditional
representation of graphs.
