# cgs

This project is a Grbl CNC gcode sender & controller.

## Directories

- cmd/: CLI interface.
- control/: Console UI control.
- gcode/: G-Code parsing & utilities.
- grbl/: Grbl communication.
- serialtcp/: connect to serial port via TCP socket.

## Code changes

- Don't bother to write tests.
- Test any changes by running `make ci`.
- If it fails, fix & rerun until complete.
