// file: internal/service/ai_service.go

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/config"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/repository"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type AIService struct {
	genaiClient *genai.GenerativeModel
}

func NewAIService() *AIService {
	ctx := context.Background()
	apiKey := config.Get("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create genai client: %v", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.7)
	return &AIService{genaiClient: model}
}

// --- FUNGSI BANTUAN BARU UNTUK MEMBERSIHKAN JSON ---
func cleanAIResponseToJSON(rawResponse genai.Text) string {
    rawStr := string(rawResponse)
    log.Printf("Respons mentah dari AI: %s", rawStr)

    // Langsung cari blok yang diawali dengan [ dan diakhiri dengan ]
    reArr := regexp.MustCompile(`(?s)\[.*\]`)
    jsonString := reArr.FindString(rawStr)

    // Jika tidak ketemu array, coba cari blok objek tunggal { ... } sebagai fallback
    if jsonString == "" {
        reObj := regexp.MustCompile(`(?s)\{.*\}`)
        jsonString = reObj.FindString(rawStr)
    }

    return jsonString
}

// GenerateRoadmapWithAI membuat roadmap berdasarkan deskripsi tujuan.
func (s *AIService) GenerateRoadmapWithAI(ctx context.Context, goalDescription string) ([]repository.RoadmapStep, error) {
	log.Println("Memanggil AI Gemini untuk membuat roadmap...")
	prompt := fmt.Sprintf(
		`Sebagai seorang productivity coach, buatkan roadmap untuk tujuan ini: "%s". 
		Berikan 3 sampai 5 langkah utama yang realistis. 
		JAWAB HANYA DENGAN FORMAT JSON ARRAY seperti ini, tanpa teks pembuka atau penutup sama sekali: 
		[{"step_order": 1, "title": "Judul Langkah 1"}, {"step_order": 2, "title": "Judul Langkah 2"}]`,
		goalDescription,
	)

	resp, err := s.genaiClient.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gagal menghasilkan konten dari AI: %w", err)
	}
	
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("respons AI tidak valid atau kosong")
	}
	
	aiResponse := resp.Candidates[0].Content.Parts[0].(genai.Text)
	
	// Gunakan helper untuk membersihkan respons
	cleanedJSON := cleanAIResponseToJSON(aiResponse)
	log.Printf("Respons Roadmap AI setelah dibersihkan: %s", cleanedJSON)
	
	var steps []repository.RoadmapStep
	if err := json.Unmarshal([]byte(cleanedJSON), &steps); err != nil {
		return nil, fmt.Errorf("gagal mem-parsing JSON dari AI: %w. Respons AI: %s", err, cleanedJSON)
	}

	return steps, nil
}

// GenerateDailyTasksWithAI membuat daftar tugas harian berdasarkan konteks.
func (s *AIService) GenerateDailyTasksWithAI(ctx context.Context, goalDesc string, currentStepTitle string, yesterdayTasks []repository.Task) ([]repository.Task, error) {
	log.Println("Memanggil AI Gemini untuk membuat jadwal harian...")

	var yesterdaySummary string
	if len(yesterdayTasks) > 0 {
		// ... (logika untuk membuat yesterdaySummary tetap sama) ...
	} else {
		yesterdaySummary = "Ini adalah hari pertama, belum ada riwayat tugas."
	}

	prompt := fmt.Sprintf(
        `Sebagai seorang productivity coach, buatkan 3-4 tugas HARI INI.
        Tujuan besar pengguna: "%s".
        FOKUS UTAMA HARI INI adalah pada langkah roadmap: "%s".
        Konteks dari kemarin: %s.

        Berdasarkan FOKUS UTAMA hari ini, berikan tugas-tugas yang sangat spesifik dan bisa dikerjakan.
        JAWAB HANYA DENGAN FORMAT JSON ARRAY seperti ini, tanpa teks tambahan:
        [{"title": "Judul Tugas Spesifik 1"}, {"title": "Judul Tugas Spesifik 2"}]`,
        goalDesc,
        currentStepTitle, // <-- Gunakan konteks baru
        yesterdaySummary,
    )

	resp, err := s.genaiClient.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gagal menghasilkan tugas harian dari AI: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("respons AI tidak valid atau kosong untuk tugas harian")
	}

	aiResponse := resp.Candidates[0].Content.Parts[0].(genai.Text)

	// Gunakan helper untuk membersihkan respons
	cleanedJSON := cleanAIResponseToJSON(aiResponse)
	log.Printf("Respons Tugas Harian AI setelah dibersihkan: %s", cleanedJSON)

	type AITask struct {
		Title string `json:"title"`
	}
	var aiTasks []AITask
	if err := json.Unmarshal([]byte(cleanedJSON), &aiTasks); err != nil {
		return nil, fmt.Errorf("gagal mem-parsing JSON dari AI untuk tugas harian: %w. Respons AI: %s", err, cleanedJSON)
	}

	var newTasks []repository.Task
	for _, t := range aiTasks {
		newTasks = append(newTasks, repository.Task{Title: t.Title})
	}

	return newTasks, nil
}


// GenerateReviewFeedback membuat feedback motivasional (TIDAK PERLU PEMBERSIH JSON).
func (s *AIService) GenerateReviewFeedback(ctx context.Context, goalDesc string, summary []repository.TaskSummary) (string, error) {
	log.Println("Memanggil AI Gemini untuk membuat feedback review yang kontekstual...")

    // --- LOGIKA BARU UNTUK MEMBUAT NARASI ---
	completedCount := 0
	missedCount := 0
    pendingCount := 0

	for _, s := range summary {
		if s.Status == "completed" {
			completedCount = s.Count
		} else if s.Status == "missed" {
			missedCount = s.Count
		} else if s.Status == "pending" {
            pendingCount = s.Count
        }
	}

    narrative := fmt.Sprintf(
        "Pengguna menyelesaikan %d tugas, melewatkan %d tugas, dan masih memiliki %d tugas yang belum selesai.",
        completedCount,
        missedCount,
        pendingCount,
    )
    // --- AKHIR LOGIKA NARASI ---

	prompt := fmt.Sprintf(
		`Anda adalah seorang productivity coach yang suportif. Tujuan besar pengguna adalah: "%s".
		Berikut adalah ringkasan performa mereka hari ini: "%s".
		Berikan feedback singkat (2-3 kalimat) yang positif dan membangun. Jika ada tugas yang selesai, puji progres mereka menuju tujuan besarnya. Jika tidak ada yang selesai, berikan semangat tanpa menghakimi untuk mencoba lagi besok.
        JAWAB SEBAGAI COACH, BUKAN SEBAGAI ASISTEN. JANGAN GUNAKAN FORMAT JSON.`,
		goalDesc,
		narrative,
	)

	resp, err := s.genaiClient.GenerateContent(ctx, genai.Text(prompt))
    if err != nil {
		return "", fmt.Errorf("gagal menghasilkan feedback dari AI: %w", err)
	}
	
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "Sepertinya ada sedikit masalah saat menghasilkan feedback, tapi tetap semangat untuk esok hari!", nil
	}

	aiResponse := resp.Candidates[0].Content.Parts[0].(genai.Text)
	return string(aiResponse), nil
}