# Nightreign Companion — Project Scope

> Ứng dụng desktop đồng hành cho **Elden Ring Nightreign** (Windows/PC), viết bằng **Go**.
> Trạng thái: **Draft v0.2** — 2026-10-09. Các mục đánh dấu ⚠️ cần xác minh hoặc chờ quyết định.
> Các quyết định đã chốt: xem [mục 8](#8-quyết-định-đã-chốt--câu-hỏi-còn-mở).

---

## 1. Mục tiêu & Nguyên tắc

### 1.1 Mục tiêu
Một app chạy song song với game, cung cấp 5 nhóm tính năng:

| # | Module | Mô tả ngắn |
|---|--------|------------|
| M1 | **Connection Checker** | Ping, jitter, packet loss tới server/peer; FPS & frametime |
| M2 | **Cycle Timer** | Đếm ngược tới lúc bo thu, Night 1, Night 2, boss cuối |
| M3 | **Relic Stats** | Tra cứu giá trị thực của chỉ số relic (vd. *Physical Attack Up +2* = bao nhiêu %) và cách cộng dồn |
| M4 | **Builds** | Duyệt build tham khảo từ nguồn + tự tạo/nhập build (optional) |
| M5 | **Boss Predictor** | Thống kê boss từng đêm; dự đoán Nightlord ở Deep of Night (depth ≥ 3) dựa trên Night 1 & Night 2 |
| M6 | **Community Data** *(sau)* | Gom dữ liệu run từ nhiều người (opt-in) để cải thiện M4 & M5 |

### 1.2 Nguyên tắc bắt buộc
1. **An toàn với Easy Anti-Cheat (EAC)** — app **không** inject DLL, **không** đọc/ghi memory của game, **không** hook DirectX. Mọi dữ liệu lấy từ: OS (network stack, ETW), input của người dùng, hoặc (tuỳ chọn) chụp màn hình. Đây là ràng buộc quan trọng nhất của dự án, vì vi phạm có thể khiến tài khoản người chơi bị ban.
2. **Offline-first** — toàn bộ dữ liệu game (relic, boss, timer) đóng gói sẵn trong app; mạng chỉ dùng để cập nhật dữ liệu / tải build.
3. **Data-driven** — số liệu game nằm trong file dữ liệu có version, tách khỏi code, để khi game patch chỉ cần cập nhật data, không cần build lại app.
4. **Nhẹ** — không làm tụt FPS game; mục tiêu < 1% CPU và < 150 MB RAM khi chạy nền.

### 1.3 Ngoài phạm vi (Non-goals)
- Không làm cheat, trainer, hay chỉnh save file.
- Không hỗ trợ console (PS/Xbox) ở v1 — chỉ module M3/M4/M5 dùng được độc lập nên có thể có bản web sau.
- Không tự động đọc trạng thái game từ memory (xem nguyên tắc 1).

---

## 2. Kiến trúc tổng thể

### 2.1 Stack đề xuất

| Lớp | Lựa chọn | Lý do |
|-----|----------|-------|
| Ngôn ngữ core | **Go 1.26** | Đã cài sẵn; tốt cho networking, concurrency, ra 1 file `.exe` |
| Desktop UI | **Wails v2** (Go backend + web frontend) | UI hiện đại bằng HTML/CSS, có cửa sổ frameless/always-on-top cho overlay, binding Go↔JS tự động |
| Frontend | **Svelte + TypeScript + Vite** | Bundle nhỏ, nhanh, dễ làm overlay |
| Lưu trữ | **SQLite** qua `modernc.org/sqlite` (pure Go, không cần CGO) | Lưu lịch sử run, build cá nhân, log mạng |
| Dữ liệu game | **JSON/YAML embed** (`go:embed`) + gói cập nhật tải về | Data-driven, có version |
| FPS | **ETW** (cơ chế giống PresentMon) | Không inject, an toàn với EAC |
| Hotkey | `golang.design/x/hotkey` | Global hotkey khi đang trong game |

> Phương án thay thế: **Fyne** (UI thuần Go) — đơn giản hơn nhưng khó làm overlay đẹp. **CLI/TUI (Bubble Tea)** — làm được rất nhanh cho bản prototype M1/M2.

### 2.2 Sơ đồ module

```
┌──────────────────────────── Frontend (Svelte) ────────────────────────────┐
│  Dashboard │ Overlay (always-on-top) │ Relics │ Builds │ Boss Predictor    │
└───────────────────────────────▲───────────────────────────────────────────┘
                                │ Wails bindings + event bus
┌───────────────────────────────┴──────────── Go backend ───────────────────┐
│ internal/netmon    ── ping/jitter/loss, phát hiện peer của game process     │
│ internal/fps       ── ETW consumer, tính FPS/frametime/1% low               │
│ internal/timer     ── state machine chu kỳ ngày/đêm, hotkey, (opt) OCR       │
│ internal/relic     ── tra cứu & tính tổng chỉ số, luật cộng dồn             │
│ internal/build     ── CRUD build, import/export, nguồn tham khảo             │
│ internal/boss      ── thống kê, mô hình dự đoán Bayes                       │
│ internal/gamedata  ── load/validate/update data pack có version             │
│ internal/store     ── SQLite repository                                     │
│ internal/config    ── settings, hotkeys, theme                              │
└────────────────────────────────────────────────────────────────────────────┘
```

### 2.3 Cấu trúc thư mục

```
NightreignCompanion/
├── main.go, app.go,      # entry Wails app (Wails yêu cầu package main ở thư mục gốc)
│   overlay.go
├── cmd/
│   └── nrc-cli/          # CLI tiện ích: validate data (sau: ping test, import run)
├── internal/             # các package ở mục 2.2 + internal/ipc
├── data/                 # data pack gốc (Go package, embed vào binary)
│   ├── manifest.json     # version data, version game tương ứng
│   ├── characters.json
│   ├── relic_effects.json
│   ├── timer_profiles.json
│   ├── bosses.json
│   └── boss_priors.json
├── frontend/             # Svelte app (dùng chung cho cửa sổ chính & overlay)
├── docs/                 # SCOPE.md, ADR, data sources
└── build/                # icon, installer config
```

### 2.4 Mô hình tiến trình (Overlay)

Wails v2 chỉ hỗ trợ **1 cửa sổ / 1 process**, nên overlay chạy như **process thứ hai** của cùng file exe:

```
nightreign-companion.exe            (process chính — giữ toàn bộ state, DB, data pack)
   │  internal/ipc: SSE trên 127.0.0.1:<port ngẫu nhiên>, xác thực bằng token ngẫu nhiên
   │  (địa chỉ + token truyền qua biến môi trường, không lộ trên command line)
   ├──► nightreign-companion.exe --overlay   (frameless, always-on-top, nền trong suốt)
   └──► nrc-fps.exe (Phase 1, chạy Admin)   (đọc ETW, gửi FPS về process chính)
```
- Overlay tự thoát khi mất kết nối với process chính quá 3 lần liên tiếp.
- Cùng kênh IPC sẽ dùng cho helper FPS chạy quyền Admin — app chính không cần Admin.

---

## 3. Chi tiết từng module

### M1 — Connection Checker

**Bài toán:** Nightreign dùng server của FromSoftware/Bandai Namco cho matchmaking, còn phiên co-op là P2P (có thể đi qua relay của Steam). Vì vậy "ping tới game" thực chất là ping tới **peer** hoặc **relay**, không phải 1 server cố định.

**Tính năng:**
| Tính năng | Cách làm | Ưu tiên |
|-----------|----------|---------|
| Ping/jitter/loss tới các endpoint cố định (Steam, gateway, DNS) | ICMP qua `pro-bing`, fallback TCP connect time | P0 |
| Phát hiện kết nối của game | Liệt kê UDP/TCP endpoint của process `nightreign.exe` qua Windows API `GetExtendedUdpTable`/`GetExtendedTcpTable` (chỉ đọc bảng mạng của OS, không đụng vào game) | P1 |
| Đo latency tới peer | Ping IP peer phát hiện được (nhiều peer chặn ICMP → hiển thị "không đo được" thay vì số sai) | P1 |
| Đánh giá chất lượng mạng nhà | Bufferbloat test, packet loss dài hạn, gợi ý (Wi-Fi vs LAN, NAT type) | P2 |
| **FPS / frametime / 1% low** | Đọc sự kiện Present từ ETW (provider DXGI/D3D9/DxgKrnl) — giống PresentMon; cần quyền Admin hoặc nhóm "Performance Log Users" | **P0** (D3) |
| Overlay mini | Góc màn hình: `Ping 45ms · Loss 0% · 60 FPS` | P1 |
| Lịch sử & biểu đồ | Lưu vào SQLite, xem lại khi run bị lag | P2 |

**Ngưỡng cảnh báo (cấu hình được):** ping > 150 ms, jitter > 30 ms, loss > 2%, FPS < 50.

⚠️ Lưu ý: Nightreign khoá 60 FPS — FPS chủ yếu để phát hiện tụt khung hình, không phải benchmark.

---

### M2 — Cycle Timer

**Bài toán:** App không đọc được trạng thái game (do EAC), nên timer phải được **kích hoạt từ bên ngoài**.

**Cách kích hoạt:**
1. **Hotkey** (P0) — người chơi bấm phím khi bắt đầu Day 1 (lúc rơi xuống từ chim). Các hotkey phụ để "sync" lại khi bước sang Day 2, hoặc pause/reset.
2. **Nhận diện màn hình (OCR/template match)** (P2, tuỳ chọn) — chụp màn hình định kỳ, nhận diện banner "Day I / Day II" hoặc thông báo bo thu để tự sync. Chỉ đọc pixel màn hình như phần mềm quay video, không tương tác với game.

**State machine của 1 expedition:**
```
DAY1_EXPLORE → DAY1_SHRINK_1 → DAY1_EXPLORE_2 → DAY1_SHRINK_2 → NIGHT1_BOSS
   → DAY2_EXPLORE → DAY2_SHRINK_1 → DAY2_EXPLORE_2 → DAY2_SHRINK_2 → NIGHT2_BOSS
   → NIGHTLORD (Day 3)
```
Night boss phase không có giới hạn thời gian → timer **dừng** và chờ hotkey "Boss xong" để bắt đầu Day 2.

**Thời lượng mỗi phase** để trong `timer_profiles.json`, mỗi chế độ (Normal / Deep of Night / sự kiện) một profile:
```json
{
  "id": "normal",
  "game_version": "x.y.z",
  "phases": [
    { "id": "DAY1_EXPLORE",  "seconds": 270, "label": "Bo bắt đầu thu lần 1" },
    { "id": "DAY1_SHRINK_1", "seconds": 180, "label": "Bo đang thu" }
  ]
}
```
⚠️ Thời lượng chính xác phải đo lại thực tế / đối chiếu tài liệu cộng đồng — số trong ví dụ là placeholder.

**UI:** overlay đếm ngược lớn + phase kế tiếp; cảnh báo âm thanh/ TTS tuỳ chọn ở mốc 60s / 30s / 10s trước khi bo thu.

---

### M3 — Relic Stats

**Bài toán:** Game chỉ ghi "Physical Attack Up +2" mà không nói giá trị. App cung cấp giá trị thực (đã datamine) và tính tổng khi trang bị nhiều relic.

**Mô hình dữ liệu (`relic_effects.json`):**
```json
{
  "id": "phys_atk_up",
  "name": { "en": "Physical Attack Up", "vi": "Tăng sát thương vật lý" },
  "category": "attack",
  "tiers": [
    { "level": 0, "value": 0.02, "unit": "multiplier" },
    { "level": 1, "value": 0.03, "unit": "multiplier" },
    { "level": 2, "value": 0.04, "unit": "multiplier" }
  ],
  "stacking": "multiplicative",
  "stacks_with_self": true,
  "character_restriction": null,
  "deep_only": false,
  "source": "regulation.bin SpEffectParam #xxxx",
  "verified_on": "x.y.z"
}
```
⚠️ Số trong ví dụ là placeholder — dữ liệu thật lấy từ datamine (`regulation.bin` qua Smithbox/DSMapStudio) và đối chiếu wiki cộng đồng; mỗi entry phải ghi **nguồn** và **game version** đã xác minh.

**Tính năng:**
- Tìm kiếm/lọc hiệu ứng theo tên, loại, nhân vật (Wylder, Guardian, Ironeye, Duchess, Raider, Revenant, Recluse, Executor, + nhân vật DLC).
- **Relic Calculator:** chọn 3 slot relic (và slot Deep relic) → hiện tổng chỉ số thực, chỉ rõ cái nào cộng dồn, cái nào không (cùng hiệu ứng có stack không, additive hay multiplicative).
- Hiển thị cả hiệu ứng **tiêu cực** (curse) của Deep of Night relic.
- So sánh 2 bộ relic.

---

### M4 — Builds

**Tính năng:**
| Tính năng | Ưu tiên |
|-----------|---------|
| Thư viện build có sẵn (curated, đóng gói trong data pack) — gắn tag: nhân vật, Nightlord mục tiêu, solo/trio, độ khó | P0 |
| Xem build: nhân vật, relic + hiệu ứng (link sang M3), vũ khí ưu tiên, passive nên nhặt, ghi chú chiến thuật | P0 |
| **Tự tạo build (manual, optional)** — form chọn nhân vật + relic + ghi chú | P1 |
| Import/Export bằng **share code** (JSON → nén → base64) để gửi qua Discord | P1 |
| Liên kết tới nguồn gốc (URL bài viết/video) | P1 |
| Đồng bộ build cộng đồng từ một repo GitHub (pull JSON) | P2 |

⚠️ **Về việc lấy build "từ nguồn":** không scrape trực tiếp các wiki/web thương mại (vi phạm ToS, dễ vỡ). Thay vào đó: build được curate thủ công vào data pack kèm link nguồn + credit tác giả, hoặc lấy từ nguồn có API/giấy phép mở.

**Nguồn tham khảo build (đề xuất):**

| Nguồn | Loại | Dùng thế nào |
|-------|------|--------------|
| **Fextralife Nightreign Wiki** | Wiki, có trang Builds + dữ liệu relic/vũ khí | Tham khảo, tóm tắt lại + link gốc (không copy nguyên văn) |
| **Game8** | Build guide, tier list theo nhân vật | Tham khảo meta, link gốc |
| **Reddit r/Nightreign** | Build & thảo luận cộng đồng, datamine relic | Curate build nổi bật, xin phép/credit tác giả |
| **YouTube creators** | Build showcase theo Nightlord | Link video trong trường `source.url` |
| **Datamine (`regulation.bin` qua Smithbox)** | Số liệu gốc | Nguồn chính cho M3, không phải build |
| **Dữ liệu cộng đồng của chính app (M6)** | Run thực tế opt-in | Dài hạn: build "meta" được xếp hạng theo win rate thật |

Hướng đi: **giai đoạn đầu curate thủ công** từ các nguồn trên (~3–5 build/nhân vật); **giai đoạn sau** M6 trở thành nguồn chính — build được đề xuất dựa trên dữ liệu run thật của người dùng.

**Format build:**
```json
{
  "id": "uuid",
  "name": "Bleed Executor",
  "character": "executor",
  "target_nightlord": ["gladius"],
  "relics": [{ "color": "red", "effects": ["phys_atk_up:2", "..."] }],
  "priority_weapons": ["..."],
  "notes": "markdown",
  "source": { "author": "...", "url": "..." },
  "schema_version": 1
}
```

---

### M5 — Boss Statistics & Nightlord Predictor

**Bài toán:** Ở chế độ thường, người chơi chọn Nightlord. Ở **Deep of Night depth ≥ 3**, Nightlord bị ẩn/ngẫu nhiên → cần đoán dựa trên các boss đã gặp ở Night 1 và Night 2 (và các dấu hiệu khác).

**Thu thập dữ liệu:**
1. **Ghi nhận run** (P0): sau mỗi run người dùng nhập nhanh (hoặc chọn từ dropdown trên overlay): chế độ, depth, Night 1 boss, Night 2 boss, Shifting Earth (nếu có), Nightlord thực tế.
2. **Dữ liệu prior** (P0): bảng pool boss theo Nightlord từ datamine/cộng đồng, đóng gói trong `boss_priors.json`.
3. **Dữ liệu cộng đồng** (P2): tuỳ chọn chia sẻ run ẩn danh lên server/GitHub để tăng cỡ mẫu.

**Mô hình dự đoán — Naive Bayes có làm mượt:**
```
P(Nightlord = L | N1 = a, N2 = b, ctx) ∝ P(L | ctx) · P(N1 = a | L, ctx) · P(N2 = b | L, ctx, N1 = a)
```
- `ctx` = depth, map event (Shifting Earth)…
- `P(L | ctx)`: prior — đều nhau nếu không có dữ liệu.
- Likelihood: từ datamine (nếu pool boss phụ thuộc Nightlord) **kết hợp** tần suất quan sát được, dùng Laplace/Dirichlet smoothing để tránh xác suất 0 khi ít mẫu.
- Nếu boss nào **chỉ xuất hiện** với một Nightlord nhất định → loại trừ cứng (xác suất = 0 cho các Nightlord khác).

**Output:** xếp hạng Nightlord kèm %, cỡ mẫu, độ tin cậy, và gợi ý chuẩn bị (điểm yếu nguyên tố, relic/build phù hợp → link sang M3/M4).
```
Night 1: Tibia Mariner · Night 2: Demi-Human Queen
→ Gladius 46% · Fulghor 22% · Libra 15% · ...   (n = 312 run)
   Gladius yếu Holy → gợi ý build: ...
```

**Thống kê:** win rate theo Nightlord/nhân vật, tần suất boss từng đêm, thời gian clear trung bình (lấy từ M2).

⚠️ Cần xác minh: pool boss đêm ở Deep of Night có thực sự phụ thuộc Nightlord hay hoàn toàn độc lập. Nếu độc lập, mô hình chỉ trả về prior (và app phải nói rõ "không có tín hiệu", không đưa ra % giả tạo).

---

### M6 — Community Data (opt-in) — *Milestone sau*

**Mục tiêu:** Gom dữ liệu run từ nhiều người để (1) tăng độ chính xác dự đoán Nightlord ở M5, (2) xếp hạng build theo hiệu quả thực tế ở M4.

**Quyền riêng tư — bắt buộc:**
- **Mặc định TẮT.** Lần đầu mở app hiện màn hình giải thích rõ: gửi dữ liệu gì, để làm gì, lưu ở đâu → người dùng chọn *Chia sẻ* / *Không chia sẻ*. Đổi lại được bất cứ lúc nào trong Settings.
- Chỉ gửi **dữ liệu gameplay**: chế độ, depth, nhân vật, boss từng đêm, Nightlord, kết quả, thời gian clear, relic/build dùng (tuỳ chọn riêng). **Không** gửi tên Steam, IP-gắn-danh-tính, dữ liệu mạng/FPS.
- Định danh bằng **ID ngẫu nhiên** sinh trên máy (không liên kết tài khoản); có nút **"Xem dữ liệu đã gửi"** và **"Xoá dữ liệu của tôi"** trên server.
- Cấp độ chia sẻ tách riêng: ☐ Boss/Nightlord · ☐ Build & relic · ☐ Thời gian clear.

**Kiến trúc (dự kiến):**
```
App (queue local trong SQLite, gửi batch khi có mạng)
   → HTTPS POST /v1/runs  (Go API server, rate-limit, validate schema)
   → PostgreSQL
   → job tổng hợp định kỳ → xuất "aggregates.json" (thống kê đã ẩn danh, đủ ngưỡng mẫu)
   → App tải aggregates.json như một phần data pack
```
- App **chỉ tải về số liệu tổng hợp**, không bao giờ tải run của người khác.
- Chống dữ liệu rác: validate tổ hợp boss hợp lệ, rate-limit theo ID, loại outlier.
- Hosting gợi ý: 1 server Go nhỏ (Fly.io / VPS) + Postgres managed; chi phí thấp ở quy mô ban đầu.

⚠️ Scope lớn → để **Phase 4**. Trong các phase trước, app vẫn lưu run **local** với schema giống hệt payload sẽ gửi lên, để khi bật M6 không phải migrate.

---

## 4. Dữ liệu game (Data Pack)

| File | Nội dung | Nguồn |
|------|----------|-------|
| `manifest.json` | version data, game version, checksum | — |
| `relic_effects.json` | hiệu ứng relic + giá trị theo tier | datamine `regulation.bin` + wiki |
| `timer_profiles.json` | thời lượng phase theo chế độ | đo thực tế + cộng đồng |
| `bosses.json` | Night boss, Nightlord, điểm yếu, ảnh icon | datamine + wiki |
| `boss_priors.json` | pool boss theo Nightlord/depth | datamine + thống kê |
| `builds/*.json` | build curated | cộng đồng (có credit) |

- Mọi file có JSON Schema; CI chạy `nrc-cli validate` để kiểm tra.
- App kiểm tra bản data mới (GitHub Releases), tải về, verify checksum, áp dụng không cần cập nhật `.exe`.
- Hỗ trợ đa ngôn ngữ: **Tiếng Việt + English** ngay từ đầu (tên trong data có dạng `{en, vi}`).

---

## 5. Lộ trình (Roadmap)

| Phase | Nội dung | Kết quả |
|-------|----------|---------|
| **0 — Nền móng** ✅ | Khung Wails + Svelte, cửa sổ chính + cửa sổ overlay, `gamedata`, `store`, `config`, `ipc`, CI (lint, test, validate data) | App rỗng chạy được |
| **1 — MVP** | M2 timer (hotkey + overlay), M3 tra cứu relic, M1 ping/loss + **FPS qua ETW** (chạy Admin) | Dùng được trong game |
| **2 — Core** | M5 ghi run (local) + dự đoán Bayes, M4 thư viện build curated + tạo build, M3 calculator | Đủ 5 module |
| **3 — Nâng cao** | Phát hiện peer, biểu đồ lịch sử, share code, OCR auto-sync timer | Bản 1.0 (base game) |
| **4 — Cộng đồng** | M6 Community Data opt-in (server + aggregates), build xếp hạng theo dữ liệu thật | Bản 1.x |
| **5 — DLC** | Data cho *The Forsaken Hollows* (nhân vật, Nightlord, relic, boss mới) | Milestone kế tiếp |

---

## 6. Rủi ro

| Rủi ro | Mức | Giảm thiểu |
|--------|-----|-----------|
| EAC coi app là đáng ngờ | Cao | Không inject/đọc memory; overlay là cửa sổ riêng; FPS qua ETW; ghi rõ trong README |
| Patch game làm sai số liệu | Trung bình | Data pack tách rời, có `verified_on`, cập nhật OTA |
| Không đo được ping peer (relay/chặn ICMP) | Trung bình | Hiển thị trung thực "N/A", đo chất lượng mạng nội bộ thay thế |
| Thiếu dữ liệu cho mô hình dự đoán | Trung bình | Prior từ datamine + smoothing + hiển thị cỡ mẫu/độ tin cậy |
| Overlay không hiện trên game Fullscreen độc quyền | Thấp | Hướng dẫn dùng Borderless Windowed |
| Bản quyền hình ảnh/tên trong game | Thấp | Dự án fan, phi thương mại, credit FromSoftware/Bandai Namco |

---

## 7. Đề xuất bổ sung (chờ trao đổi)

1. **Run Tracker / Lịch sử run** — gộp dữ liệu từ M2 (thời gian) + M5 (boss) thành nhật ký mỗi run: nhân vật, kết quả, thời gian clear. Là nền dữ liệu cho thống kê.
2. **Map Helper** — ghi chú vị trí theo Shifting Earth (Crater, Mountaintop, Rotted Woods, Noklateo…) và điểm đánh dấu quan trọng; có thể dùng seed map của cộng đồng.
3. **Weakness Cheat Sheet** — bảng điểm yếu/kháng của mọi Night boss & Nightlord, hiện tự động trên overlay khi chọn boss ở M5.
4. **Party Mode** — 3 người cùng dùng app chia sẻ timer & dự đoán qua phòng LAN/WebSocket.
5. **Discord Rich Presence** — hiển thị Nightlord đang đánh, Day hiện tại.

---

## 8. Quyết định đã chốt & câu hỏi còn mở

### Đã chốt (2026-10-09)
| # | Chủ đề | Quyết định |
|---|--------|-----------|
| D1 | UI | Desktop GUI (Wails) **có overlay** ngay từ đầu, không làm bản CLI/TUI |
| D2 | Timer | **Hotkey** cho MVP; OCR auto-sync ở Phase 3 |
| D3 | FPS | Đo FPS qua ETW là **cần thiết** → app yêu cầu quyền Admin (đưa vào Phase 1) |
| D4 | Dữ liệu cộng đồng | **Có**, gom từ nhiều người, **opt-in rõ ràng** → module M6, để Phase 4; trước đó lưu local cùng schema |
| D5 | Nguồn build | Curate thủ công từ Fextralife / Game8 / Reddit / YouTube (link + credit); dài hạn dùng dữ liệu M6 |
| D6 | DLC | **Không** trong v1 — để Phase 5 |

### Còn mở
1. **Admin toàn app hay chỉ phần FPS?** Đề xuất: app chính chạy quyền thường, tách một helper nhỏ `nrc-fps.exe` chạy Admin chỉ để đọc ETW (an toàn hơn, UAC chỉ hỏi khi bật FPS).
2. **Tên hiển thị / branding** — giữ "Nightreign Companion"? (lưu ý tránh dùng logo chính thức của game).
3. **Phát hành** — open-source trên GitHub? (có lợi cho việc cộng đồng đóng góp data pack & build).
