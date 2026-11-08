# Gyro Breakout in Go (TinyGo)

[Sample gameplay video](output.mp4)

A bare-metal Breakout game for the STM32F429I-DISCO, written in Go and built with TinyGo. It uses the board's LCD, gyroscope, blue user button, and onboard ST-LINK.

## Play

Keep the board flat and still while the calibration message is displayed.

For normal play, leave the blue button released when calibration finishes. Rotate the board to move the paddle. The gyro measures rotation rate, so the paddle stops when rotation stops. Press the button to launch the ball; during play, press or hold it to fire. A quick rotation on the other axis changes the ball speed in 0.1× steps, from 0.5× to 1.3×. Return the board to rest between gestures.

Each brick takes two hits from the ball or a fireball: it turns red on the first hit and disappears on the second. The bonus brick starts blue and turns red when damaged. It adds two lives if the ball destroys it; a fireball destroys it without the bonus. Clear every brick to win. The elapsed round time appears below the result. After a win or loss, release and press the button to restart, then release and press again to launch.

To start an automatic simulation, hold the blue button when calibration finishes. The controller plays repeated games without fireballs, initially aiming to open a center column and reach above the bricks. Each round starts at 1.0× speed; use the same rotation gesture as normal play to change it. The result remains visible briefly between rounds. Reset the board to return to normal play.

## Build and flash

You need Go, TinyGo, GNU Make, `arm-none-eabi-size` (from the ARM embedded binutils), OpenOCD, and an STM32F429I-DISCO. The Go module declares its minimum Go version in `go.mod`. TinyGo must be compatible with the installed Go version and provide the STM32F407 runtime used by this target.

```sh
make test    # Host tests
make check   # Host tests, go vet, and TinyGo firmware build
make build   # Build build/gyro-breakout-go.elf and print its size
make flash   # Build, program, verify, and reset through ST-LINK
```

`make check` runs locally and does not flash the board. Running `make` cleans, builds, and flashes the board. `make clean` removes build output. The Go build helper uses `tinygo` from PATH; set the `TINYGO` environment variable to use another executable. The `.vscode` directory contains an optional Cortex-Debug configuration.



 # The firmware’s text, data, and BSS sizes

```
   text    data     bss     dec     hex filename
  21368    1832  153864  177064   2b3a8 build/gyro-breakout-go.elf
```

### Tested setup and board checks

The reported board is an STM32F429I-DISCO, MB1075 B-01. The toolchain used for the local build is TinyGo 0.44.0, Go 1.27.3, and LLVM 20.1.8. Other combinations have not been verified. The game has been reported working on the board; the subsequent simulation, elapsed-time, two-hit brick, and timer-fault changes have not yet received a complete board check.

Before a release, check on the board:

- Startup, calibration message, and normal gyro and button controls.
- A win and a loss, including elapsed time and restart.
- Simulation selection, repeated rounds, and reset back to normal play.
- The error indicator when practical to test; host checks cannot verify peripheral faults.

## Design and target limitations

Game state, physics, rendering, and the platform interface live in the root package. `cmd/firmware` contains the LCD, gyroscope, and register-level board code; `targets` contains the memory layout and interrupt table. The build helper in `cmd/build` combines the local target with the installed TinyGo runtime.

The target uses the F429's 2 MiB flash and 192 KiB main SRAM layout and a project-local F429 interrupt table. It currently reuses TinyGo's F407 runtime and machine definitions, then switches to a 16 MHz internal clock for the board drivers. It is not a complete upstream F429 board port. The LCD uses a single framebuffer without vertical synchronization, so tearing is possible.

Host tests cover game logic and gyro filtering, but do not exercise the register-level hardware drivers or validate timing on the board. A successful OpenOCD verification confirms the image was programmed; check display, calibration, controls, and gameplay on the board as well.

## Troubleshooting

- If controls drift, reset and keep the board still during calibration.
- A red LED or red screen indicates a startup or sensor failure; reset and retry.
- If OpenOCD cannot connect, check the ST-LINK USB connection and the `board/stm32f429discovery.cfg` configuration.

## License

See [LICENSE](LICENSE).
