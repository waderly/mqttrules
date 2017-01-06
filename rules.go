package mqttrules

import (
	"encoding/json"

	"github.com/Knetic/govaluate"
	log "github.com/Sirupsen/logrus"
	"github.com/oliveagle/jsonpath"
)

type Action struct {
	Topic   string
	Payload string
	QoS     byte
	Retain  bool
}

type Rule struct {
	Trigger             string
	Schedule            string
	Condition           string
	Actions             []Action
	conditionExpression *govaluate.EvaluableExpression
}

var rules = map[string]Rule{}

func (c *client) handleIncomingRule(ruleset string, rule string, value string) {
	log.Infof("Received rule '%s/%s'", ruleset, rule)

	var r Rule
	err := json.Unmarshal([]byte(value), &r)
	if err != nil {
		log.Errorf("Unable to parse JSON string: %v", err)
		return
	}

	if len(r.Condition) > 0 {
		r.conditionExpression, err = govaluate.NewEvaluableExpression(r.Condition)
		if err != nil {
			log.Errorf("Error parsing rule condition: %v", err)
			return
		}
	}

	if c.rules != nil && len(c.rules[ruleset][rule].Trigger) > 0 {
		c.RemoveRuleSubscription(r.Trigger, ruleset, rule)
	}

	_, exists := c.rules[ruleset]
	if !exists {
		c.rules[ruleset] = make(map[string]Rule)
	}
	c.rules[ruleset][rule] = r

	if len(r.Trigger) > 0 {
		c.AddRuleSubscription(r.Trigger, ruleset, rule)
	}
	log.Infof("Added rule %s: %+v\n", rule, r)

	//TODO

}

/* public functions */

func (c *client) AddRuleSubscription(topic string, ruleset string, rule string) {
	if !c.ensureSubscription(topic) {
		return
	}

	_, exists := c.subscriptions[topic].rules[ruleset]
	if !exists {
		c.subscriptions[topic].rules[ruleset] = make(map[string]bool)
	}
	c.subscriptions[topic].rules[ruleset][rule] = true
}

func (c *client) RemoveRuleSubscription(topic string, ruleset string, rule string) {
	_, exists := c.subscriptions[topic]
	if c.mqttClient == nil || !exists {
		return
	}

	delete(c.subscriptions[topic].rules[ruleset], rule)
	if len(c.subscriptions[topic].rules[ruleset]) == 0 {
		delete(c.subscriptions[topic].rules, ruleset)
	}
	c.contemplateUnsubscription(topic)
}

func (c *client) ExecuteRule(ruleset string, rule string, triggerPayload string) {
	log.Infof("Executing rule %s/%s", ruleset, rule)

	r := c.rules[ruleset][rule]
	if len(r.Condition) > 0 {
		fPayload := func(args ...interface{}) (interface{}, error) {
			if len(args) == 0 {
				// No JSON path given - return whole payload
				return triggerPayload, nil
			}
			var jsonData interface{}
			err := json.Unmarshal([]byte(triggerPayload), &jsonData)
			if err != nil {
				log.Errorf("JSON parsing error in trigger payload when executing rule %s/%s: %v", ruleset, rule, err)
				return triggerPayload, err
			}
			res, err := jsonpath.JsonPathLookup(jsonData, args[0].(string))
			if err != nil {
				log.Errorf("JSON lookup error in trigger payload when executing rule %s/%s: %v", ruleset, rule, err)
				return triggerPayload, err
			}

			return res, nil
		}

		functions := map[string]govaluate.ExpressionFunction{
			"payload": fPayload,
		}

		expression, err := govaluate.NewEvaluableExpressionWithFunctions(r.Condition, functions)
		if err != nil {
			log.Errorln("Error parsing condition:", err)
			return
		}
		result, err := expression.Evaluate(c.parameterValues)
		if err != nil {
			log.Errorln("Error evaluating condition:", err)
			return
		}
		if result != true {
			log.Debugln("Condition evaluated to false, rule not executed")
			return
		}
	}
	for _, r := range r.Actions {
		// TODO: Replace parameters in payload
		c.Publish(r.Topic, r.QoS, r.Retain, r.Payload)
	}
}
