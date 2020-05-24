## Das \[Keyboard\]

This repository tries to re-implement the wire-protocol used by the *Das
Keyboard 4Q*. The keyboard comes with a software providing a [REST
API](https://www.daskeyboard.io/api-resources/signal/resource-description/),
which is relatively easy to use. Unfortunately it also introduces significant
delay – there is about a one second delay until the LED on the keyboard lights
up. This is likely due to the fact that the software wants to be a
"notification center", which shows notification windows in the graphical user
interface. By talking to the keyboard directly, this implementation aims to
provide significantly faster latency. Judging from USB traces, latency in the
order of 10&nbsp;ms should be feasible.

A similar implementation (in TypeScript) exists for the 5Q model at
[diefarbe/node-lib](https://github.com/diefarbe/node-lib). Despite the similar
product names, wire-protocols appear to be entirely different.

## License

Licensed under the [terms of the ISC license](LICENSE).
