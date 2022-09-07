module Elevator

go 1.17

require Driver-go v0.0.0
replace Driver-go => ./Driver-go

require Network-go v0.0.0
replace Network-go => ./Network-go

require HallAssigner v0.0.0
replace HallAssigner => ./HallAssigner

require ConfigsAndTypes v0.0.0
replace ConfigsAndTypes => ./ConfigsAndTypes

require OrderHandler v0.0.0
replace OrderHandler => ./OrderHandler

require FSM v0.0.0
replace FSM => ./FSM

require Requests v0.0.0
replace Requests => ./Requests