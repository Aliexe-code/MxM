package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aliexe/blockChain/internal/blockchain"
)

type MinerCLI struct {
	bc         *blockchain.Blockchain
	isMining   bool
	minerID    string
	difficulty int
	stats      *MiningStats
	statsMu    sync.RWMutex
}

type MiningStats struct {
	BlocksMined   int
	TotalRewards  float64
	StartTime     time.Time
	LastBlockTime time.Time
	AverageTime   time.Duration
}

const (
	DefaultMinerID      = "default-miner"
	DefaultDifficulty   = 2
	StatsUpdateInterval = 5 * time.Second
)

func main() {
	cli := &MinerCLI{
		difficulty: DefaultDifficulty,
		minerID:    DefaultMinerID,
		stats:      &MiningStats{},
	}

	// Parse command line flags
	flag.StringVar(&cli.minerID, "miner", DefaultMinerID, "Miner identifier")
	flag.IntVar(&cli.difficulty, "difficulty", DefaultDifficulty, "Mining difficulty (1-8)")

	flag.Parse()

	if len(flag.Args()) == 0 {
		cli.showHelp()
		return
	}

	command := strings.ToLower(flag.Args()[0])

	switch command {
	case "start":
		cli.startMining()
	case "stop":
		cli.stopMining()
	case "status":
		cli.showStatus()
	case "stats":
		cli.showDetailedStats()
	case "set-difficulty":
		cli.setDifficulty()
	case "help", "--help", "-h":
		cli.showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		cli.showHelp()
	}
}

func (cli *MinerCLI) showHelp() {
	fmt.Println("🔥 CryptoChain Mining CLI")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  miner [COMMAND] [OPTIONS]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  start           Start mining blocks")
	fmt.Println("  stop            Stop mining process")
	fmt.Println("  status          Show current mining status")
	fmt.Println("  stats           Show detailed mining statistics")
	fmt.Println("  set-difficulty  Set mining difficulty")
	fmt.Println("  help            Show this help message")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -miner string        Miner identifier (default \"default-miner\")")
	fmt.Println("  -difficulty int      Mining difficulty 1-8 (default 2)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  miner start -miner alice -difficulty 3")
	fmt.Println("  miner status")
	fmt.Println("  miner set-difficulty 4")
	fmt.Println("  miner stop")
}

func (cli *MinerCLI) startMining() {
	if cli.isMining {
		fmt.Println("⚠️  Mining is already running")
		return
	}

	cli.bc = blockchain.NewBlockchain()
	cli.isMining = true
	cli.stats.StartTime = time.Now()

	fmt.Printf("🚀 Starting mining with difficulty %d for miner %s\n", cli.difficulty, cli.minerID)
	fmt.Println("Press Ctrl+C to stop mining...")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start stats display goroutine
	go cli.displayStatsPeriodically()

	// Mining loop
	blockCount := 0
	for cli.isMining {
		select {
		case <-sigChan:
			cli.stopMining()
			return
		default:
			// Mine a new block
			blockData := fmt.Sprintf("Block #%d mined by %s", blockCount+1, cli.minerID)

			duration, err := cli.bc.AddBlockWithMining(blockData, cli.minerID, cli.difficulty)

			if err != nil {
				log.Printf("❌ Error mining block: %v", err)
				continue
			}

			if duration > 0 {
				blockCount++
				cli.updateStats(duration)
				cli.showMiningSuccess(blockCount, duration)
			} else {
				fmt.Printf("⏳ Mining attempt failed, retrying...\n")
			}

			// Small delay to prevent excessive CPU usage
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (cli *MinerCLI) stopMining() {
	if !cli.isMining {
		fmt.Println("⚠️  Mining is not running")
		return
	}

	cli.isMining = false
	fmt.Println("\n🛑 Mining stopped")

	if cli.stats.BlocksMined > 0 {
		cli.showMiningSummary()
	}
}

func (cli *MinerCLI) showStatus() {
	if cli.isMining {
		cli.statsMu.RLock()
		uptime := time.Since(cli.stats.StartTime)
		blocksMined := cli.stats.BlocksMined
		totalRewards := cli.stats.TotalRewards
		averageTime := cli.stats.AverageTime
		cli.statsMu.RUnlock()

		fmt.Printf("🔥 Mining Status: ACTIVE\n")
		fmt.Printf("⛏️  Miner: %s\n", cli.minerID)
		fmt.Printf("🎯 Difficulty: %d\n", cli.difficulty)
		fmt.Printf("⏱️  Uptime: %v\n", uptime.Round(time.Second))
		fmt.Printf("📊 Blocks Mined: %d\n", blocksMined)
		fmt.Printf("💰 Total Rewards: %.2f\n", totalRewards)
		if averageTime > 0 {
			fmt.Printf("⚡ Average Block Time: %v\n", averageTime.Round(time.Millisecond))
		}
	} else {
		fmt.Println("🔥 Mining Status: STOPPED")
	}
}

func (cli *MinerCLI) showDetailedStats() {
	if cli.bc == nil {
		fmt.Println("❌ No blockchain data available. Start mining first.")
		return
	}

	stats := cli.bc.GetMiningStats()

	fmt.Println("📊 Detailed Mining Statistics")
	fmt.Println("============================")

	// Blockchain stats
	fmt.Printf("📦 Total Blocks: %v\n", stats["total_blocks"])
	fmt.Printf("💰 Total Rewards: %.2f\n", stats["total_rewards"].(float64))
	fmt.Printf("🏆 Reward Count: %v\n", stats["reward_count"])

	// Miner stats
	if minerRewards, ok := stats["miner_rewards"].(map[string]float64); ok {
		fmt.Println("\n👥 Miner Rewards:")
		for miner, reward := range minerRewards {
			fmt.Printf("  💎 %s: %.2f\n", miner, reward)
		}
	}

	if minerBlocks, ok := stats["miner_blocks"].(map[string]int); ok {
		fmt.Println("\n🏗️  Miner Block Counts:")
		for miner, blocks := range minerBlocks {
			fmt.Printf("  📦 %s: %d blocks\n", miner, blocks)
		}
	}
}

func (cli *MinerCLI) setDifficulty() {
	if len(flag.Args()) < 2 {
		fmt.Println("❌ Please provide a difficulty value (1-8)")
		fmt.Println("Usage: miner set-difficulty <difficulty>")
		return
	}

	newDifficulty, err := strconv.Atoi(flag.Args()[1])
	if err != nil {
		fmt.Printf("❌ Invalid difficulty: %v\n", err)
		return
	}

	if newDifficulty < 1 || newDifficulty > 8 {
		fmt.Println("❌ Difficulty must be between 1 and 8")
		return
	}

	cli.difficulty = newDifficulty
	fmt.Printf("✅ Mining difficulty set to %d\n", newDifficulty)

	if cli.isMining {
		fmt.Printf("⚠️  New difficulty will apply to the next block\n")
	}
}

func (cli *MinerCLI) updateStats(duration time.Duration) {
	cli.statsMu.Lock()
	defer cli.statsMu.Unlock()

	cli.stats.BlocksMined++
	cli.stats.LastBlockTime = time.Now()

	// Update total rewards from blockchain
	if cli.bc != nil {
		cli.stats.TotalRewards = cli.bc.GetMinerRewards(cli.minerID)
	}

	// Calculate average time
	if cli.stats.BlocksMined == 1 {
		cli.stats.AverageTime = duration
	} else {
		totalTime := cli.stats.AverageTime*time.Duration(cli.stats.BlocksMined-1) + duration
		cli.stats.AverageTime = totalTime / time.Duration(cli.stats.BlocksMined)
	}
}

func (cli *MinerCLI) showMiningSuccess(blockCount int, duration time.Duration) {
	reward := cli.bc.CalculateReward(cli.difficulty)
	fmt.Printf("✅ Block #%d mined in %v (Reward: %.2f)\n",
		blockCount, duration.Round(time.Millisecond), reward)
}

func (cli *MinerCLI) showMiningSummary() {
	cli.statsMu.RLock()
	blocksMined := cli.stats.BlocksMined
	totalRewards := cli.stats.TotalRewards
	startTime := cli.stats.StartTime
	averageTime := cli.stats.AverageTime
	cli.statsMu.RUnlock()

	uptime := time.Since(startTime)
	fmt.Println("\n📈 Mining Summary")
	fmt.Println("=================")
	fmt.Printf("⏱️  Total Uptime: %v\n", uptime.Round(time.Second))
	fmt.Printf("📦 Blocks Mined: %d\n", blocksMined)
	fmt.Printf("💰 Total Rewards: %.2f\n", totalRewards)
	if averageTime > 0 {
		fmt.Printf("⚡ Average Block Time: %v\n", averageTime.Round(time.Millisecond))
		fmt.Printf("🚀 Mining Rate: %.2f blocks/hour\n",
			float64(blocksMined)/uptime.Hours())
	}
}

func (cli *MinerCLI) displayStatsPeriodically() {
	ticker := time.NewTicker(StatsUpdateInterval)
	defer ticker.Stop()

	for {
		<-ticker.C

		cli.statsMu.RLock()
		isMining := cli.isMining
		blocksMined := cli.stats.BlocksMined
		totalRewards := cli.stats.TotalRewards
		startTime := cli.stats.StartTime
		averageTime := cli.stats.AverageTime
		cli.statsMu.RUnlock()

		if !isMining {
			return
		}

		if blocksMined > 0 {
			uptime := time.Since(startTime)
			fmt.Printf("\n📊 [Live] Blocks: %d | Rewards: %.2f | Uptime: %v | Avg Time: %v\n",
				blocksMined,
				totalRewards,
				uptime.Round(time.Second),
				averageTime.Round(time.Millisecond))
		}
	}
}
