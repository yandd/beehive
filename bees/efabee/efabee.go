/*
 *    Copyright (C) 2014-2018 Christian Muehlhaeuser
 *
 *    This program is free software: you can redistribute it and/or modify
 *    it under the terms of the GNU Affero General Public License as published
 *    by the Free Software Foundation, either version 3 of the License, or
 *    (at your option) any later version.
 *
 *    This program is distributed in the hope that it will be useful,
 *    but WITHOUT ANY WARRANTY; without even the implied warranty of
 *    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *    GNU Affero General Public License for more details.
 *
 *    You should have received a copy of the GNU Affero General Public License
 *    along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 *    Authors:
 *      Christian Muehlhaeuser <muesli@gmail.com>
 */

// Package efabee is a Bee that interfaces with the public EVA API.
package efabee

import (
	"time"

	"github.com/muesli/goefa"

	"github.com/muesli/beehive/bees"
)

// EFABee is a Bee that interfaces with the public EVA API.
type EFABee struct {
	bees.Bee

	Provider string
	efa      *goefa.Provider

	eventChan chan bees.Event
}

// Action triggers the action passed to it.
func (mod *EFABee) Action(action bees.Action) []bees.Placeholder {
	outs := []bees.Placeholder{}

	switch action.Name {
	case "directions":
		var originParam, destParam string
		action.Options.Bind("origin", &originParam)
		action.Options.Bind("destination", &destParam)

		origin, err := mod.efa.FindStop(originParam)
		if err != nil {
			mod.Logln("Origin does not exist or name is not unique!")
			return outs
		}
		mod.Logf("Selected origin: %s (%d)", origin[0].Name, origin[0].ID)

		destination, err := mod.efa.FindStop(destParam)
		if err != nil {
			mod.Logln("Destination does not exist or name is not unique!")
			return outs
		}
		mod.Logf("Selected destination: %s (%d)", destination[0].Name, destination[0].ID)

		routes, err := mod.efa.Route(origin[0].ID, destination[0].ID, time.Now())
		for _, route := range routes {
			mod.Logf("Trip duration: %s, %d transfers\n", route.ArrivalTime.Sub(route.DepartureTime), len(route.Trips)-1)
			for _, trip := range route.Trips {
				origin, err := mod.efa.Stop(trip.OriginID)
				if err != nil {
					mod.LogErrorf("%s", err)
					return outs
				}
				dest, err := mod.efa.Stop(trip.DestinationID)
				if err != nil {
					mod.LogErrorf("%s", err)
					return outs
				}

				ev := bees.Event{
					Bee:  mod.Name(),
					Name: "trip",
					Options: []bees.Placeholder{
						{
							Name:  "mottype",
							Type:  "string",
							Value: trip.MeansOfTransport.MotType.String(),
						},
						{
							Name:  "arrival_time",
							Type:  "string",
							Value: trip.ArrivalTime.Format("15:04"),
						},
						{
							Name:  "departure_time",
							Type:  "string",
							Value: trip.DepartureTime.Format("15:04"),
						},
						{
							Name:  "route",
							Type:  "string",
							Value: trip.MeansOfTransport.Number,
						},
						{
							Name:  "origin",
							Type:  "string",
							Value: origin.Name,
						},
						{
							Name:  "destination",
							Type:  "string",
							Value: dest.Name,
						},
						{
							Name:  "origin_platform",
							Type:  "string",
							Value: trip.OriginPlatform,
						},
						{
							Name:  "destination_platform",
							Type:  "string",
							Value: trip.DestinationPlatform,
						},
					},
				}
				mod.eventChan <- ev
			}

			// only post one trip for now
			break
		}

	case "departures":
		stop := ""
		amount := 3
		action.Options.Bind("stop", &stop)
		action.Options.Bind("amount", &amount)

		//FIXME get departures
		station, err := mod.efa.FindStop(stop)
		if err != nil {
			mod.Logln("Stop does not exist or name is not unique!")
			return outs
		}
		mod.Logf("Selected stop: %s (%d)", station[0].Name, station[0].ID)

		departures, err := station[0].Departures(time.Now(), amount)
		if err != nil {
			mod.Logln("Could not retrieve departure times!")
			return outs
		}
		for _, departure := range departures {
			destStop, err := mod.efa.Stop(departure.ServingLine.DestStopID)
			if err != nil {
				mod.Logln("Could not retrieve destination stop")
				return outs
			}

			mod.Logf("Route %-5s due in %-2d minute%s --> %s",
				departure.ServingLine.Number,
				departure.CountDown,
				"s",
				destStop.Name)

			ev := bees.Event{
				Bee:  mod.Name(),
				Name: "departure",
				Options: []bees.Placeholder{
					{
						Name:  "mottype",
						Type:  "string",
						Value: departure.ServingLine.MotType.String(),
					},
					{
						Name:  "eta",
						Type:  "int",
						Value: departure.CountDown,
					},
					{
						Name:  "etatime",
						Type:  "string",
						Value: departure.DateTime.Format("15:04"),
					},
					{
						Name:  "route",
						Type:  "string",
						Value: departure.ServingLine.Number,
					},
					{
						Name:  "destination",
						Type:  "string",
						Value: destStop.Name,
					},
				},
			}
			mod.eventChan <- ev
		}

	default:
		panic("Unknown action triggered in " + mod.Name() + ": " + action.Name)
	}

	return outs
}

// Run executes the Bee's event loop.
func (mod *EFABee) Run(eventChan chan bees.Event) {
	mod.eventChan = eventChan
}

// ReloadOptions parses the config options and initializes the Bee.
func (mod *EFABee) ReloadOptions(options bees.BeeOptions) {
	mod.SetOptions(options)

	options.Bind("provider", &mod.Provider)
	mod.efa = goefa.NewProvider(mod.Provider, true)
}
