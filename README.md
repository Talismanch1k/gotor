
## Communication with peer

Here is a scheme that describes how communication with a peer proceeds.

```
Client                         Peer
  |                              |
  | -------- handshake --------> |
  | <------- handshake ----------|
  |                              |
  | <-------- bitfield ----------|  peer says what piece it has
  | -------- interested -------->|  we say that we need some
  |                              |
  | <--------- unchoke ----------|  peer has allowed data requests
  |                              |
  | -------- request ----------->|  piece=7, begin=0, length=16384
  | -------- request ----------->|  piece=7, begin=16384, length=16384
  | -------- request ----------->|  piece=7, begin=32768, length=16384
  |                              |
  | <--------- piece ------------|  send block piece=7, begin=0
  | <--------- piece ------------|  send block piece=7, begin=16384
  | <--------- piece ------------|  send block piece=7, begin=32768
  |                              |
  |      put together piece      |
  |      check SHA-1             |
  |                              |
  | -------- have(7) ----------->|  inform that we have piece 7
```

## To-do

- [ ] Daemon for torrent downloading
- [ ] Rarest first piece picking
- [ ] Limiting goroutine number
