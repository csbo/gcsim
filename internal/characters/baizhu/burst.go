
package baizhu

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/geometry"
	"github.com/genshinsim/gcsim/pkg/core/player"
	"github.com/genshinsim/gcsim/pkg/modifier"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/core/reactions"
)

var burstFrames []int

const burstHitmark = 12
const tickTaskDelay = 20

func init() {
	burstFrames = frames.InitAbilSlice(123) // Q -> N1
	burstFrames[action.ActionSkill] = 122   // Q -> E
	burstFrames[action.ActionDash] = 122    // Q -> D
	burstFrames[action.ActionJump] = 122    // Q -> J
	burstFrames[action.ActionSwap] = 120    // Q -> Swap
}

func (c *char) Burst(p map[string]int) action.ActionInfo {
	// dmg
	// ai := combat.AttackInfo{
	// 	ActorIndex:       c.Index,
	// 	Abil:             "Yuqi",
	// 	AttackTag:        attacks.AttackTagElementalBurst,
	// 	ICDTag:           attacks.ICDTagNone,
	// 	ICDGroup:         attacks.ICDGroupDefault,
	// 	StrikeType:       attacks.StrikeTypeDefault,
	// 	Element:          attributes.Dendro,
	// 	Durability:       25,
	// 	Mult:             burst[c.TalentLvlBurst()],
	// }
	//snap := c.Snapshot(&ai)
	burstArea := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), geometry.Point{Y: 1.5}, 10)
	//c.Core.QueueAttackWithSnap(
	//	ai,
	//	snap,
	//	combat.NewCircleHitOnTarget(burstArea.Shape.Pos(), nil, 4.5),
	//	burstHitmark,
	//)

	d := c.createBurstSnapshot()
	
	c.ConsumeEnergy(5)

	// make sure that this task gets executed:
	// - after q initial hit hitlag happened
	// - before sayu can get affected by any more hitlag
	c.QueueCharTask(func() {
		// first tick is at 145
		for i := 150; i < 14 * 60 + 150; i += 150 {
			c.Core.Tasks.Add(func() {
				// check for player
				// only check HP of active character
				char := c.Core.Player.ActiveChar()

				// check for enemy
				enemy := c.Core.Combat.ClosestEnemyWithinArea(burstArea, nil)

				// determine whether to attack or heal
				// - C1 makes burst able to attack both an enemy and heal the player at the same time
				d.Pattern = combat.NewCircleHitOnTarget(enemy, nil, 4)
				c.Core.QueueAttackEvent(d, 0)

				c.Core.Player.Heal(player.HealInfo{
					Caller:  c.Index,
					Target:  char.Index,
					Message: "Yuqi",
					Src:     (bursthealpp[c.TalentLvlSkill()]*c.MaxHP() + bursthealflat[c.TalentLvlSkill()]),
					Bonus:   d.Snapshot.Stats[attributes.Heal],
				})

				c.Core.Player.ActiveChar().AddReactBonusMod(character.ReactBonusMod{
					Base: modifier.NewBase("Yuqi-bloom", 360),
					Amount: func(ai combat.AttackInfo) (float64, bool) {
						switch ai.AttackTag {
						case attacks.AttackTagBloom:
						case attacks.AttackTagHyperbloom:
						case attacks.AttackTagBurgeon:
						default:
							return 0, false
						}
	
						hp := c.MaxHP()
						if hp > 50000 {
							hp = 50000
						}
						return hp / 1000 * 0.02, false
					},
				})

				c.Core.Player.ActiveChar().AddReactBonusMod(character.ReactBonusMod{
					Base: modifier.NewBase("Yuqi-quicken", 360),
					Amount: func(ai combat.AttackInfo) (float64, bool) {
						if ai.Catalyzed && ai.CatalyzedType == reactions.Aggravate {
							hp := c.MaxHP()
							if hp > 50000 {
								hp = 50000
							}
							return hp / 1000 * 0.008, false
						}
						return 0, false
					},
				})
			}, i)
		}
	}, tickTaskDelay)

	c.SetCD(action.ActionBurst, 20*60)

	return action.ActionInfo{
		Frames:          frames.NewAbilFunc(burstFrames),
		AnimationLength: burstFrames[action.InvalidAction],
		CanQueueAfter:   burstFrames[action.ActionDash], // earliest cancel
		State:           action.BurstState,
	}
}

// TODO: is this helper function needed?
func (c *char) createBurstSnapshot() *combat.AttackEvent {
	ai := combat.AttackInfo{
		ActorIndex: c.Index,
		Abil:       "Yuqi",
		AttackTag:  attacks.AttackTagElementalBurst,
		ICDTag:     attacks.ICDTagElementalBurst,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Dendro,
		Durability: 25,
		Mult:       burst[c.TalentLvlBurst()],
	}
	snap := c.Snapshot(&ai)
	ae := combat.AttackEvent{
		Info:        ai,
		SourceFrame: c.Core.F,
		Snapshot:    snap,
	}
	return &ae
}
