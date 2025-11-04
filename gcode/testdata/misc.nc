G28.1         ; Set home position to current location
G30.1         ; Set secondary home position to current location
G38.2 Z-10 F100 ; Probe toward workpiece (stop on contact, error if not triggered)
G38.3 Z-5 F100  ; Probe away from workpiece (stop on contact, error if not triggered)
G38.4 Z-2 F100  ; Probe toward workpiece (stop on contact, no error if not triggered)
G38.5 Z0 F100   ; Probe away from workpiece (stop on contact, no error if not triggered)
G92.1        ; Cancel coordinate offset
