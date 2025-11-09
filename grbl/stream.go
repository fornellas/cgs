package grbl

// TODO Streaming Protocol: Character-Counting
//   EEPROM Issues: can't stream with character counting for some commands:
//     write commands: G10 L2, G10 L20, G28.1, G30.1, $x=, $I=, $Nx=, $RST=
//     read commands: G54-G59, G28, G30, $$, $I, $N, $#
//   Check g-code with $C before sending
