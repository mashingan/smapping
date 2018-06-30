![license](https://img.shields.io/github/license/mashape/apistatus.svg?style=plastic)

# smapping
Golang structs generic mapping.

## Motivation
Working with between ``struct``, and ``json`` with **Golang** has various
degree of difficulty.
The thing that makes difficult is that sometimes we get arbitrary ``json``
or have to make json with arbitrary fields.  
In **Golang** we can achieve that using ``interface{}`` that act as ``"Any"``
value.
While it seems good, however ``interface{}`` only gives us ability to *disable* 
type propagation and various reasoning.
It's a trade-off that one has to make to enable interfacing with dynamically
defined such as ``json``.


## LICENSE
MIT
