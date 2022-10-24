package metric_gen

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sanity-io/litter"
)

const (
	stringType = iota
	setStringType
	hardCodedOne
	valueType
	intDimensionType
	ipType

	allPlacement
	TCPPlacement
	HTTPPlacement
	NONEPlacement

	// 65536 in regular AVR
	maxMessageLength = 32768
)

type Generator struct {
	interval time.Duration
	cfg      Config

	messageCache []*Message
	cacheLock    sync.Mutex

	iterator int

	setMetrics     bool
	setMetricsSets []string

	simple bool
}

type Config struct {
	UniquePercent       int
	DimensionSize       int
	MetricSetsPerMinute int
}

type Message struct {
	Message    string
	MetricSets int
}

type Option func(generator *Generator)

func WithFixedMetricSets(set ...string) Option {
	return func(generator *Generator) {
		generator.setMetrics = true
		generator.setMetricsSets = set
	}
}

func WithSimpleSet(simple bool) Option {
	return func(generator *Generator) {
		generator.simple = simple
	}
}

func NewGenerator(cfg Config, options ...Option) (*Generator, error) {
	g := &Generator{
		cfg:          cfg,
		messageCache: make([]*Message, cfg.MetricSetsPerMinute),
	}
	metricSetsPerSecond := g.cfg.MetricSetsPerMinute / 60

	for _, o := range options {
		o(g)
	}

	fmt.Println("Preparing fixed metric cache for easier access by the generator")
	g.messageCache = g.buildMessageCache(metricSetsPerSecond)
	fmt.Printf("Initial cache of size %v prepared\n", len(g.messageCache))

	return g, nil
}

func (g *Generator) Generate(ctx context.Context, output chan *Message) error {
	fmt.Printf("Generation simple: %v, Settings: %s\n", g.simple, litter.Sdump(g.cfg))

	now := time.Now()
	fmt.Printf("%s - Generator started\n", now.Format("15:04:05"))

	metricSetsPerSecond := g.cfg.MetricSetsPerMinute / 60
	fmt.Printf("Generating %v metric sets per second\n", metricSetsPerSecond)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	fullIntervalTimer := time.NewTicker(time.Minute)
	defer fullIntervalTimer.Stop()

	ctx, cancelRegeneration := context.WithCancel(context.Background())
	go func() {
		timer := time.NewTimer(time.Minute)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				g.regenerateCache()
				timer.Reset(time.Minute)
			}
		}
	}()

	var sentMetricSetsInThisMinute int
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping generator")
			cancelRegeneration()
			return nil
		case <-ticker.C:
			var sentMetricSetsThisSecond int
			for sentMetricSetsThisSecond < metricSetsPerSecond {
				msg := g.getMessageFromCache()
				sentMetricSetsThisSecond += msg.MetricSets
				sentMetricSetsInThisMinute += msg.MetricSets

				output <- msg
			}
		case <-fullIntervalTimer.C:
			now := time.Now()
			fmt.Printf("%s - generated/expected: %v/%v\n",
				now.Format("15:04:05"), sentMetricSetsInThisMinute, g.cfg.MetricSetsPerMinute)
			sentMetricSetsInThisMinute = 0
		}
	}
}

func (g *Generator) getMessageFromCache() *Message {
	g.cacheLock.Lock()
	defer g.cacheLock.Unlock()

	g.iterator++
	if g.iterator >= len(g.messageCache) {
		g.iterator = 0
	}

	return g.messageCache[g.iterator]
}

func (g *Generator) buildMessageCache(messagesPerSecond int) []*Message {
	var cache []*Message
	for i := 1; i <= 60; i++ {
		if i%12 == 0 {
			now := time.Now()
			fmt.Printf("%s - Generating cache %v%%\n", now.Format("15:04:05"), (i/6)*10)
		}

		cache = append(cache, g.buildMessages(messagesPerSecond)...)
	}
	return cache
}

func (g *Generator) buildMessages(howManyMetricSets int) []*Message {
	var result []*Message

	msg := &Message{}
	for i := 0; i < howManyMetricSets; i++ {
		value, _ := rand.Int(rand.Reader, big.NewInt(100))
		unique := int(value.Int64())+1 <= g.cfg.UniquePercent
		metricSet := g.makeMetricSet(unique) + ";"

		if len(metricSet)+len(msg.Message) > maxMessageLength {
			result = append(result, msg)
			msg = &Message{}
		}

		msg.MetricSets++
		msg.Message += metricSet
	}
	result = append(result, msg)

	return result
}

func (g *Generator) makeMetricSet(uniqueDimension bool) string {
	if g.setMetrics {

		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(g.setMetricsSets))))
		return g.setMetricsSets[r.Int64()]
	}

	// choose one string dimension that will be unique
	uniqueDimPositionBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(fieldOrder))))
	for fieldOrder[uniqueDimPositionBig.Int64()].Type != stringType {
		uniqueDimPositionBig, _ = rand.Int(rand.Reader, big.NewInt(int64(len(fieldOrder))-8))
	}
	uniqueDimPosition := int(uniqueDimPositionBig.Int64())

	var placement int
	placementBigRand, _ := rand.Int(rand.Reader, big.NewInt(3))
	placementRand := int(placementBigRand.Int64())
	if placementRand <= 1 {
		placement = HTTPPlacement
	} else {
		placement = TCPPlacement
	}

	var result string
	for i, field := range fieldOrder {
		// if current field is not of the placement we want
		// write a separator and go to next
		if field.Placement != allPlacement && field.Placement != placement {
			result += " "
			continue
		}

		if uniqueDimension && i == uniqueDimPosition {
			// if random dimension is not a string dimension choose next
			if field.Type != stringType {
				uniqueDimPosition++
			} else {
				result += "\"rnd" + fastRndString(g.cfg.DimensionSize) + "\" "
				continue
			}
		}

		if g.simple {
			if len(field.SimpleTestPossibilities) == 0 {
				result += " "
				continue
			}

			randomPickBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(field.SimpleTestPossibilities))))
			randomPick := int(randomPickBig.Int64())
			fieldVal := field.SimpleTestPossibilities[randomPick]
			if field.Type == intDimensionType || field.Type == valueType || field.Type == hardCodedOne {
				val, err := strconv.Atoi(fieldVal)
				if err != nil {
					fmt.Println(err.Error())
					result += "0 "
					continue
				}
				result += strconv.FormatInt(int64(val), 16)
				result += " "
				continue
			} else if field.Type == ipType {
				result += fieldVal + " "
				continue
			}
			result = result + "\"" + fieldVal + "\"" + " "
			continue
		}

		switch field.Type {
		case stringType:
			// this is a way of increasing the size of dimensions while keeping them semi-unique and consistent
			randBase, _ := rand.Int(rand.Reader, big.NewInt(int64(field.Size)))
			randN := int(randBase.Int64()) + int(math.Pow10(g.cfg.DimensionSize))
			result += "\"" + field.Name + strconv.Itoa(randN) + "\""
		case valueType:
			result += strconv.FormatInt(time.Now().Unix()/10, 16)
		case ipType:
			result += "0100007fffff00000000000000000000"
		case hardCodedOne:
			result += "1"
		case intDimensionType:
			randBase, _ := rand.Int(rand.Reader, big.NewInt(int64(field.Size)))
			randN := int(randBase.Int64()) + int(math.Pow10(g.cfg.DimensionSize))
			result += strconv.FormatInt(int64(randN), 16)
		case setStringType:
			randomPickBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(field.Possibilities))))
			randomPick := int(randomPickBig.Int64())
			result += "\"" + field.Possibilities[randomPick] + "\""
		}
		result += " "
	}

	result = strings.TrimSuffix(result, " ")
	return result
}

func (g *Generator) regenerateCache() {
	now := time.Now()
	fmt.Printf("%s - Starting regenerating cache\n", now.Format("15:04:05"))
	newCache := g.buildMessageCache(g.cfg.MetricSetsPerMinute / 60)
	g.cacheLock.Lock()
	g.messageCache = newCache
	g.cacheLock.Unlock()
	now = time.Now()
	fmt.Printf("%s - New cache generated and replaced\n", now.Format("15:04:05"))
}
