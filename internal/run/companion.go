package run

import (
	"context"
	"fmt"
	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/event"
	"github.com/hectorgimenez/koolo/internal/pather"
	"github.com/hectorgimenez/koolo/internal/town"
	"time"
)

type Companion struct {
	baseRun
}

func (s Companion) Name() string {
	return "Companion"
}

func (s Companion) BuildActions() []action.Action {
	var lastEntranceEntered = time.Now()
	var leaderUnitIDTarget data.UnitID
	// TODO: Deregister this listener or will leak
	s.EventListener.Register(func(ctx context.Context, e event.Event) error {
		if evt, ok := e.(event.CompanionLeaderAttackEvent); ok && config.Characters[evt.Supervisor()].CharacterName == s.CharacterCfg.Companion.LeaderName {
			leaderUnitIDTarget = evt.TargetUnitID
		}

		return nil
	})

	return []action.Action{
		action.NewChain(func(d data.Data) []action.Action {
			leaderRosterMember, leaderFound := d.Roster.FindByName(s.CharacterCfg.Companion.LeaderName)
			if !leaderFound {
				s.logger.Warn(fmt.Sprintf("Leader not found: %s", s.CharacterCfg.Companion.LeaderName))
				return []action.Action{}
			}

			// Leader is NOT in the same act
			if leaderRosterMember.Area.Act() != d.PlayerUnit.Area.Act() {
				_, foundPortal := getClosestPortal(d)

				// Follower is NOT in town
				if !d.PlayerUnit.Area.IsTown() {

					// Portal is found nearby
					if foundPortal {
						return []action.Action{
							s.builder.UsePortalInTown(),
						}
					}

					// Portal is not found nearby
					if hasEnoughPortals(d) {
						return []action.Action{
							s.builder.ReturnTown(),
						}
					}

					// there is NO portal open and follower does NOT have enough portals. Just exit
					return []action.Action{}
				}

				// Follower is in town. Just change the act
				return []action.Action{
					s.builder.WayPoint(town.GetTownByArea(leaderRosterMember.Area).TownArea()),
				}
			}

			// Is leader too far away?
			if pather.DistanceFromMe(d, leaderRosterMember.Position) > 100 {
				// In some cases this "follower in town -> use portal -> follower outside town -> use portal"
				// loop can go on forever. But it is responsibility of a leader to not cause it...

				_, foundPortal := getClosestPortal(d)

				// Follower in town
				if d.PlayerUnit.Area.IsTown() {
					if foundPortal {
						return []action.Action{
							s.builder.UsePortalInTown(),
						}
					}

					// Go to TP waiting area
					return []action.Action{
						s.builder.MoveTo(func(d data.Data) (data.Position, bool) {
							tpArea := town.GetTownByArea(d.PlayerUnit.Area).TPWaitingArea(d)
							return tpArea, true
						}),
					}
				}

				// If we have an entrance, use it
				entrance, entranceFound := getClosestEntrances(d)
				if entranceFound && leaderRosterMember.Area == entrance.Area && time.Since(lastEntranceEntered) > (time.Second*4) {
					lastEntranceEntered = time.Now()

					return []action.Action{
						s.builder.MoveToArea(entrance.Area),
					}
				}

				// If we have portal open, use it
				if foundPortal {
					return []action.Action{
						s.builder.UsePortalInTown(),
					}
				}

				// Otherwise just wait
				return []action.Action{
					s.builder.Wait(100),
				}
			}

			// If distance leader is at acceptable distance and is attacking, support him
			distanceFromMe := pather.DistanceFromMe(d, leaderRosterMember.Position)
			if distanceFromMe < 30 {
				_, found := d.Monsters.FindByID(leaderUnitIDTarget)
				if s.CharacterCfg.Companion.Attack && found {
					return []action.Action{
						s.char.KillMonsterSequence(func(d data.Data) (data.UnitID, bool) {
							return leaderUnitIDTarget, true
						}, nil),
					}
				}

				// If there is no monster to attack, and we are close enough to the leader just wait
				if distanceFromMe < 4 {
					return []action.Action{
						s.builder.Wait(100),
					}
				}
			}

			return []action.Action{
				action.NewStepChain(func(d data.Data) []step.Step {
					return []step.Step{step.MoveTo(s.CharacterCfg, leaderRosterMember.Position, step.WithTimeout(time.Millisecond*500))}
				}),
			}
		}, action.RepeatUntilNoSteps()),
	}
}

func getClosestPortal(d data.Data) (*data.Object, bool) {
	for _, o := range d.Objects {
		if o.IsPortal() && pather.DistanceFromMe(d, o.Position) <= 20 {
			return &o, true
		}
	}

	return nil, false
}

func hasEnoughPortals(d data.Data) bool {
	portalTome, pFound := d.Items.Find(item.TomeOfTownPortal, item.LocationInventory)
	if pFound {
		return portalTome.Stats[stat.Quantity].Value > 0
	}

	return false
}

func getClosestEntrances(d data.Data) (*data.Level, bool) {
	for _, l := range d.AdjacentLevels {
		distFromMe := pather.DistanceFromMe(d, l.Position)
		if distFromMe <= 20 {
			return &l, true
		}
	}

	return nil, false
}
