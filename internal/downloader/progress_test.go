package downloader

import (
	"strings"
	"testing"
)

func TestProgressParser_ParseLine(t *testing.T) {
	parser := NewProgressParser()

	tests := []struct {
		name            string
		line            string
		wantPercent     float64
		wantSpeed       string
		wantEta         string
		wantStatus      string
		wantDestination string
	}{
		{
			name:            "basic progress with percent",
			line:            "[download] 10.5% of 50.00MiB at 2.50MiB/s ETA 00:20",
			wantPercent:     10.5,
			wantSpeed:       "2.50MiB/s",
			wantEta:         "00:20",
			wantStatus:      "[download]",
			wantDestination: "",
		},
		{
			name:            "percent without download prefix is ignored",
			line:            "50.0% of 100.00MiB at 5.00MiB/s ETA 00:10",
			wantPercent:     0,
			wantSpeed:       "5.00MiB/s",
			wantEta:         "00:10",
			wantStatus:      "",
			wantDestination: "",
		},
		{
			name:            "percent inside destination filename is ignored",
			line:            "[download] Destination: /home/user/Music/We Are 99% Human.f137.mp4",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "/home/user/Music/We Are 99% Human.f137.mp4",
		},
		{
			name:            "stderr warning with percent is ignored",
			line:            "WARNING: [youtube] 75% of comments parsed, retrying",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "",
		},
		{
			name:            "percent only",
			line:            "[download] 25%",
			wantPercent:     25.0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "[download]",
			wantDestination: "",
		},
		{
			name:            "speed only",
			line:            "[download] Downloading at 1.5MiB/s",
			wantPercent:     0,
			wantSpeed:       "1.5MiB/s",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "",
		},
		{
			name:            "ETA with hours",
			line:            "[download] 5% of 1.00GiB at 1.00MiB/s ETA 01:30:45",
			wantPercent:     5.0,
			wantSpeed:       "1.00MiB/s",
			wantEta:         "01:30:45",
			wantStatus:      "[download]",
			wantDestination: "",
		},
		{
			name:            "destination line mp4",
			line:            "[download] Destination: /path/to/video.mp4",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "/path/to/video.mp4",
		},
		{
			name:            "destination line webm",
			line:            "[download] Destination: /path/to/video.webm",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "/path/to/video.webm",
		},
		{
			name:            "destination line m4a audio",
			line:            "[download] Destination: /path/to/audio.m4a",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "/path/to/audio.m4a",
		},
		{
			name:            "destination line mp3",
			line:            "[download] Destination: /path/to/audio.mp3",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "/path/to/audio.mp3",
		},
		{
			name:            "empty line",
			line:            "",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "",
		},
		{
			name:            "random non-download line",
			line:            "[info] Checking video availability",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "",
		},
		{
			name:            "percent with KiB/s",
			line:            "[download]   0.8% of  109.70MiB at   400KiB/s ETA 23:13",
			wantPercent:     0.8,
			wantSpeed:       "400KiB/s",
			wantEta:         "23:13",
			wantStatus:      "[download]",
			wantDestination: "",
		},
		{
			name:            "percent with MB/s",
			line:            "[download] 60% of 100.00MiB at 1.2MB/s",
			wantPercent:     60.0,
			wantSpeed:       "1.2MB/s",
			wantEta:         "",
			wantStatus:      "[download]",
			wantDestination: "",
		},
		{
			name:            "format id line",
			line:            "[info] format: 248",
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "",
		},
		{
			name:            "progress with format id",
			line:            "[download] 30% of 50.00MiB at 2.00MiB/s format 248 ETA 00:10",
			wantPercent:     30.0,
			wantSpeed:       "2.00MiB/s",
			wantEta:         "00:10",
			wantStatus:      "[download] format 248",
			wantDestination: "",
		},
		{
			name:            "merger line captures merged destination",
			line:            `[Merger] Merging formats into "/home/user/downloads/My Video.mp4"`,
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "/home/user/downloads/My Video.mp4",
		},
		{
			name:            "ffmpeg extract audio destination",
			line:            `[ffmpeg] Destination: /home/user/downloads/My Song.mp3`,
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "/home/user/downloads/My Song.mp3",
		},
		{
			name:            "ExtractAudio destination",
			line:            `[ExtractAudio] Destination: /home/user/downloads/My Song.mp3`,
			wantPercent:     0,
			wantSpeed:       "",
			wantEta:         "",
			wantStatus:      "",
			wantDestination: "/home/user/downloads/My Song.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPercent, gotSpeed, gotEta, gotStatus, gotDestination := parser.ParseLine(tt.line)
			if gotPercent != tt.wantPercent {
				t.Errorf("ParseLine() percent = %v, want %v", gotPercent, tt.wantPercent)
			}
			if gotSpeed != tt.wantSpeed {
				t.Errorf("ParseLine() speed = %v, want %v", gotSpeed, tt.wantSpeed)
			}
			if gotEta != tt.wantEta {
				t.Errorf("ParseLine() eta = %v, want %v", gotEta, tt.wantEta)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("ParseLine() status = %v, want %v", gotStatus, tt.wantStatus)
			}
			if gotDestination != tt.wantDestination {
				t.Errorf("ParseLine() destination = %v, want %v", gotDestination, tt.wantDestination)
			}
		})
	}
}

func TestProgressParser_ReadPipeMultiLineCapturesLastDestination(t *testing.T) {
	tests := []struct {
		name      string
		lines     string
		wantFinal string
	}{
		{
			name: "audio extraction: intermediate m4a overwritten by ffmpeg mp3",
			lines: "" +
				"[download] Destination: /tmp/audio.m4a\r" +
				"[download]   0.0% of  5.00MiB at 500KiB/s ETA 00:10\r" +
				"[download] 100% of  5.00MiB at 2.00MiB/s ETA 00:00\r" +
				"[ffmpeg] Destination: /tmp/audio.mp3\r",
			wantFinal: "/tmp/audio.mp3",
		},
		{
			name: "video+audio merge: intermediate overwritten by merger mp4",
			lines: "" +
				"[download] Destination: /tmp/video.f137.mp4\r" +
				"[download] 100% of 50.00MiB at 5.00MiB/s ETA 00:00\r" +
				"[download] Destination: /tmp/video.f140.m4a\r" +
				"[download] 100% of  5.00MiB at 2.00MiB/s ETA 00:00\r" +
				`[Merger] Merging formats into "/tmp/video.mp4"` + "\r",
			wantFinal: "/tmp/video.mp4",
		},
		{
			name: "remux: webm downloaded then merged to mp4",
			lines: "" +
				"[download] Destination: /tmp/video.webm\r" +
				"[download] 100% of 30.00MiB at 3.00MiB/s ETA 00:00\r" +
				`[Merger] Merging formats into "/tmp/video.mp4"` + "\r",
			wantFinal: "/tmp/video.mp4",
		},
		{
			name: "single stream: destination never changes",
			lines: "" +
				"[download] Destination: /tmp/video.mp4\r" +
				"[download]   0.0% of 50.00MiB at 5.00MiB/s ETA 00:30\r" +
				"[download] 100% of 50.00MiB at 5.00MiB/s ETA 00:00\r",
			wantFinal: "/tmp/video.mp4",
		},
		{
			name: "ExtractAudio variant",
			lines: "" +
				"[download] Destination: /tmp/song.m4a\r" +
				"[download] 100% of  8.00MiB at 4.00MiB/s ETA 00:00\r" +
				"[ExtractAudio] Destination: /tmp/song.mp3\r",
			wantFinal: "/tmp/song.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewProgressParser()
			pipe := strings.NewReader(tt.lines)

			var lastDestination string
			sendProgress := func(_ float64, _, _, _, destination string) {
				if destination != "" {
					lastDestination = destination
				}
			}

			parser.ReadPipe(pipe, sendProgress)

			if lastDestination != tt.wantFinal {
				t.Fatalf("last destination = %q, want %q", lastDestination, tt.wantFinal)
			}
		})
	}
}
