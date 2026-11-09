# README-Screenshot img/snapshot1.png — nur über `task screenshot` starten,
# der Task legt .demo/ mit Binary und Demo-Daten (img/demo/) frisch an.
Output .demo/snapshot.gif

Set Shell "bash"
Set FontSize 20
Set Width 1400
Set Height 250
Set Padding 16
Set Theme "Dracula"

Hide
# Eigenes HOME: echte ~/.local/state/qdrover bleibt unberührt.
Type "export HOME=$PWD/.demo/home && cd .demo/project && clear && ../qdrover"
Enter
Sleep 1.5s
Show
# Vor jedem Tastendruck, sonst verschwindet die Version oben rechts.
Sleep 300ms
Screenshot img/snapshot1.png
Hide
Type "q"
