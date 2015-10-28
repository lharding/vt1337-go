# VT1337

I like playing around with game programming, especially when learning a new programming language, but it gets tedious rewriting the same code to put a window on the screen and render sprites in it, and similarly tedious figuring out yet another game library.

My solution is VT1337: a game engine built like a terminal emulator - when invoked, it runs a program, reads a series of commands from that program's `stdout` stream and uses them to render a sprite-based game scene using OpenGL, while feeding user input back to that program via its `stdin`. 

This way, it's possible to write your game logic in whatever language is nicest without worrying about whether that language has a decent sprite library, OpenGL/OpenAL bindings, etc. It also brings the bonus of inherently putting your game logic on a separate core from rendering, since it's in an entirely separate process.

## Performance?

Isn't decoding all those textual commands slow? 

Nope. Some statistics from a test run on my creaky old ThinkPad T400 (the slowest machine I could find at the moment), running `test.py` with ~1000 active sprites:

```
Queue depth: 943. Render time: 22.993091ms. Decode time: 10.448µs.
Queue depth: 851. Render time: 20.698499ms. Decode time: 10.539µs.
Queue depth: 879. Render time: 21.773303ms. Decode time: 10.383µs.
Queue depth: 938. Render time: 26.538723ms. Decode time: 12.484µs.
```

Times are per-frame, averaged over 120 frames. Translated, that's about 50FPS (thanks to the ancient GPU), and it's decoding at just under 100,000,000 commands/sec. Queue depth is the number of commands pending per-frame; the test program sends 1001 commands per frame.

More detail about the test machine:
```
⮀ lspci | grep VGA
01:00.0 VGA compatible controller: Advanced Micro Devices, Inc. [AMD/ATI] RV620/M82 [Mobility Radeon HD 3450/3470]
⮀ cat /proc/cpuinfo | grep 'model name'
model name	: Intel(R) Core(TM)2 Duo CPU     T9400  @ 2.53GHz
model name	: Intel(R) Core(TM)2 Duo CPU     T9400  @ 2.53GHz
```

## Installation & Invokation

I've checked a built binary into the repository; if you have reasonably current GL libraries installed it should just run. To build a new one:

```
cd vt1337-go
go get
go build
./vt1337-go test.py
```

I've included a very simple and stupid test program called `test.py` to demonstrate the currently-implemented features, you could of course use anything that can read from `stdin` and write to `stdout`.

## WIP

This is hardcore WIP territory. So far command decoding and proceessing, a simple camera system, and user input handling are finished, but all sprites are rendered as grey rectangles covering their dimensions.

### Roadmap:

- Better code organization (multiple files, at least)
- Sprite texture and tile-map support
- Built-in text-rendering tileset with extras for making roguelikes look nice
- Sound
- Joystick support
- Client program control over window size
- State dump/load support
