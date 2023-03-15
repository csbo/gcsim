package baizhu

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/targets"
	"github.com/genshinsim/gcsim/pkg/core/player"
)

var skillFrames []int

const (
	skillHitmark   = 33
	particleICDKey = "baizhu-particle-icd"
)

func init() {
	skillFrames = frames.InitAbilSlice(49)
	skillFrames[action.ActionBurst] = 48
	skillFrames[action.ActionDash] = 30
	skillFrames[action.ActionJump] = 32
	skillFrames[action.ActionSwap] = 48
}

func (c *char) Skill(p map[string]int) action.ActionInfo {
	ai := combat.AttackInfo{
		Abil:       "Taisu",
		ActorIndex: c.Index,
		AttackTag:  attacks.AttackTagElementalArt,
		ICDTag:     attacks.ICDTagElementalArt,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Dendro,
		Durability: 25,
		Mult:       skill[c.TalentLvlSkill()],
	}

	c.Core.QueueAttack(
		ai,
		combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 4.5),
		0,
		skillHitmark,
		c.particleCB,
	)

	stats, _ := c.Stats()
	c.Core.Tasks.Add(func() {
		c.Core.Player.Heal(player.HealInfo{
			Caller:  c.Index,
			Target:  -1,
			Message: "Taisu",
			Src:     skillhealflat[c.TalentLvlBurst()] + skillhealpp[c.TalentLvlBurst()]*c.MaxHP(),
			Bonus:   stats[attributes.Heal],
		})
	}, 77)

	c.SetCD(action.ActionSkill, 600)

	return action.ActionInfo{
		Frames:          frames.NewAbilFunc(skillFrames),
		AnimationLength: skillFrames[action.InvalidAction],
		CanQueueAfter:   skillFrames[action.ActionDash], // earliest cancel
		State:           action.SkillState,
	}
}

func (c *char) particleCB(a combat.AttackCB) {
	if a.Target.Type() != targets.TargettableEnemy {
		return
	}
	if c.StatusIsActive(particleICDKey) {
		return
	}
	c.AddStatus(particleICDKey, 0.3*60, false)

	count := 3.0
	if c.Core.Rand.Float64() < 0.5 {
		count = 4
	}
	c.Core.QueueParticle(c.Base.Key.String(), count, attributes.Dendro, c.ParticleDelay)
}