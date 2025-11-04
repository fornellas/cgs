g28.1         ; Set home position to current location
g30.1         ; Set secondary home position to current location
g38.2 z-10 f100 ; Probe toward workpiece (stop on contact, error if not triggered)
g38.3 z-5 f100  ; Probe away from workpiece (stop on contact, error if not triggered)
g38.4 z-2 f100  ; Probe toward workpiece (stop on contact, no error if not triggered)
g38.5 z0 f100   ; Probe away from workpiece (stop on contact, no error if not triggered)
g92.1        ; Cancel coordinate offset
