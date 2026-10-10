# Nightreign Companion — Project Scope

> Ứng dụng desktop đồng hành cho **Elden Ring Nightreign** (Windows/PC), viết bằng **Go**.
> Trạng thái: **Draft v0.3** — 2026-10-10. Các mục đánh dấu ⚠️ cần xác minh hoặc chờ quyết định.
> Các quyết định đã chốt: xem [mục 8](#8-quyết-định-đã-chốt--câu-hỏi-còn-mở).
>
> **Thay đổi so với v0.2:** bỏ module **Builds** (M4) vì không có nguồn dữ liệu build dùng được; thứ tự milestone sau M1 đổi thành **Boss (M5) → Relic (M3) → Timer (M2)**.

---

## 1. Mục tiêu & Nguyên tắc

### 1.1 Mục tiêu
Một app chạy song song với game, cung cấp 4 nhóm tính năng (+1 module cộng đồng về sau):

| # | Module | Mô tả ngắn | Trạng thái |
|---|--------|------------|-----------|
| M1 | **Connection Checker** | Ping, jitter, packet loss; FPS & frametime | Đang làm (Phase 1) |
| M5 | **Boss Predictor** | Thống kê boss từng đêm; dự đoán Nightlord ở Deep of Night (depth ≥ 3) dựa trên Night 1 & Night 2 | Phase 2 |
| M3 | **Relic Stats** | Tra cứu giá trị thực của chỉ số relic (vd. *Physical Attack Up +2* = bao nhiêu %) và cách cộng dồn | Phase 3 |
| M2 | **Cycle Timer** | Đếm ngược tới lúc bo thu, Night 1, Night 2, boss cuối | Phase 4 |
| M6 | **Community Data** *(sau)* | Gom dữ liệu run từ nhiều người (opt-in) để cải thiện M5 | Phase 6 |
| ~~M4~~ | ~~**Builds**~~ | **Đã bỏ** — xem [mục 3, M4](#m4--builds--đã-bỏ) | — |

> Mã module (M1…M6) giữ nguyên như v0.2 để không lệch với code/commit cũ; bảng trên xếp theo thứ tự làm.

### 1.2 Nguyên tắc bắt buộc
1. **An toàn với Easy Anti-Cheat (EAC)** — app **không** inject DLL, **không** đọc/ghi memory của game, **không** hook DirectX. Mọi dữ liệu lấy từ: OS (network stack, ETW), input của người dùng, hoặc (tuỳ chọn) chụp màn hình. Đây là ràng buộc quan trọng nhất của dự án, vì vi phạm có thể khiến tài khoản người chơi bị ban.
2. **Offline-first** — toàn bộ dữ liệu game (relic, boss, timer) đóng gói sẵn trong app; mạng chỉ dùng để cập nhật dữ liệu.
3. **Data-driven** — số liệu game nằm trong file dữ liệu có version, tách khỏi code, để khi game patch chỉ cần cập nhật data, không cần build lại app.
4. **Nhẹ** — không làm tụt FPS game; mục tiêu < 1% CPU và < 150 MB RAM khi chạy nền.

### 1.3 Ngoài phạm vi (Non-goals)
- Không làm cheat, trainer, hay chỉnh save file.
- Không hỗ trợ console (PS/Xbox) ở v1 — chỉ module M3/M5 dùng được độc lập nên có thể có bản web sau.
- Không tự động đọc trạng thái game từ memory (xem nguyên tắc 1).
- **Không làm thư viện build / build planner** (bỏ từ v0.3 — không có nguồn dữ liệu có API hoặc giấy phép mở, và scrape wiki thương mại vi phạm ToS).

---

## 2. Kiến trúc tổng thể

### 2.1 Stack đề xuất

| Lớp | Lựa chọn | Lý do |
|-----|----------|-------|
| Ngôn ngữ core | **Go 1.26** | Đã cài sẵn; tốt cho networking, concurrency, ra 1 file `.exe` |
| Desktop UI | **Wails v2** (Go backend + web frontend) | UI hiện đại bằng HTML/CSS, có cửa sổ frameless/always-on-top cho overlay, binding Go↔JS tự động |
| Frontend | **Svelte + TypeScript + Vite** | Bundle nhỏ, nhanh, dễ làm overlay |
| Lưu trữ | **SQLite** qua `modernc.org/sqlite` (pure Go, không cần CGO) | Lưu lịch sử run, log mạng |
| Dữ liệu game | **JSON/YAML embed** (`go:embed`) + gói cập nhật tải về | Data-driven, có version |
| FPS | **ETW** (cơ chế giống PresentMon) | Không inject, an toàn với EAC |
| Hotkey | `golang.design/x/hotkey` | Global hotkey khi đang trong game |

> Phương án thay thế: **Fyne** (UI thuần Go) — đơn giản hơn nhưng khó làm overlay đẹp. **CLI/TUI (Bubble Tea)** — làm được rất nhanh cho bản prototype.

### 2.2 Sơ đồ module

```
┌──────────────────────────── Frontend (Svelte) ────────────────────────────┐
│  Dashboard │ Overlay (always-on-top) │ Boss Predictor │ Relics │ Timer     │
└───────────────────────────────▲───────────────────────────────────────────┘
                                │ Wails bindings + event bus
┌───────────────────────────────┴──────────── Go backend ───────────────────┐
│ internal/netmon    ── ping/jitter/loss, (sau) phát hiện kết nối của game    │
│ internal/fps       ── ETW consumer, tính FPS/frametime/1% low               │
│ internal/boss      ── thống kê, mô hình dự đoán Bayes                       │
│ internal/relic     ── tra cứu & tính tổng chỉ số, luật cộng dồn             │
│ internal/timer     ── state machine chu kỳ ngày/đêm, hotkey, (opt) OCR       │
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
│   └── nrc-cli/          # CLI tiện ích: validate data, ping test (sau: import run)
├── internal/             # các package ở mục 2.2 + internal/ipc
├── data/                 # data pack gốc (Go package, embed vào binary)
│   ├── manifest.json     # version data, version game tương ứng
│   ├── characters.json
│   ├── bosses.json
│   ├── boss_priors.json
│   ├── relic_effects.json
│   └── timer_profiles.json
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
   └──► nightreign-companion.exe --fps-helper   (chạy Admin qua UAC; đọc ETW, POST FPS về process chính)
```
- Overlay tự thoát khi mất kết nối với process chính quá 3 lần liên tiếp.
- Helper FPS dùng cùng file exe (1 file duy nhất để phát hành), chỉ helper chạy quyền Admin — app chính không cần.
  UAC không truyền biến môi trường nên địa chỉ + token IPC đi qua tham số dòng lệnh; tên tiến trình game được validate chặt (`[A-Za-z0-9._-]+.exe`).
- Helper gửi dữ liệu bằng `POST /messages`, nhận lệnh dừng qua SSE, và tự thoát khi mất kết nối với process chính.

---

## 3. Chi tiết từng module

Các module được trình bày theo **thứ tự làm**: M1 → M5 → M3 → M2 → M6.

### M1 — Connection Checker

**Bài toán:** Nightreign dùng server của FromSoftware/Bandai Namco cho matchmaking, còn phiên co-op là P2P (có thể đi qua relay của Steam). Vì vậy "ping tới game" thực chất là ping tới **peer** hoặc **relay**, không phải 1 server cố định.

**Hiện trạng (2026-10-10):**
- ✅ Đã đo: ping/jitter/loss tới các mục tiêu cố định (mặc định `gateway`, `1.1.1.1`, `8.8.8.8`, tối đa 6 mục tiêu) — tức là **chất lượng mạng của máy/nhà mạng**, chưa phải kết nối thực của Nightreign.
- ✅ Đã đo: FPS / frametime / 1% low của `nightreign.exe` qua ETW (helper Admin), hiện cả trên overlay.
- ❌ Chưa làm: phát hiện và đo kết nối thực của game (peer / relay / server matchmaking).

**Tính năng:**
| Tính năng | Cách làm | Ưu tiên |
|-----------|----------|---------|
| ✅ Ping/jitter/loss tới các endpoint cố định (gateway, DNS, host tuỳ chọn) | ICMP qua `IcmpSendEcho` (iphlpapi — không cần Admin), TCP connect time cho mục tiêu `host:port`; cửa sổ trượt 60 mẫu + tổng phiên | P0 |
| ✅ **FPS / frametime / 1% low** | Sự kiện `Present_Start` (ID 42) của provider Microsoft-Windows-DXGI qua phiên ETW real-time — giống PresentMon; lọc theo PID của `nightreign.exe`; cửa sổ 30s (FPS 1s, trung bình, 1% low, frametime tệ nhất) | **P0** (D3) |
| ✅ Overlay mini | Góc màn hình: ping/loss từng mục tiêu + FPS | P1 |
| Phát hiện kết nối của game | **TCP:** `GetExtendedTcpTable` cho ra địa chỉ remote (server matchmaking). **UDP:** `GetExtendedUdpTable` **chỉ có cổng local, không có địa chỉ remote** (UDP không có kết nối) → để thấy peer/relay phải dùng ETW provider `Microsoft-Windows-Kernel-Network` (sự kiện gửi/nhận UDP kèm PID + địa chỉ đích), chạy trong helper Admin đã có. Chỉ đọc dữ liệu của OS, không đụng vào game | P1 |
| Đo latency tới peer/relay | Ping IP phát hiện được ở trên. Nhiều peer/relay chặn ICMP → hiển thị "không đo được" thay vì số sai | P1 |
| Đánh giá chất lượng mạng nhà | Bufferbloat test, packet loss dài hạn, gợi ý (Wi-Fi vs LAN, NAT type) | P2 |
| Lịch sử & biểu đồ | Lưu vào SQLite, xem lại khi run bị lag | P2 |

**Ngưỡng cảnh báo (cấu hình được):** ping > 150 ms, jitter > 30 ms, loss > 2%, FPS < 50.
Mức đánh giá: **Kém** khi vượt ngưỡng · **Cảnh báo** khi vượt 2/3 ngưỡng hoặc có mất gói · **Tốt** còn lại.
**Chẩn đoán:** so sánh mục tiêu `gateway` (router) với các mục tiêu Internet để phân biệt lỗi mạng nội bộ (Wi-Fi/LAN) với lỗi nhà mạng.

⚠️ Lưu ý: Nightreign khoá 60 FPS — FPS chủ yếu để phát hiện tụt khung hình, không phải benchmark.
⚠️ Cần xác minh: Nightreign đi P2P trực tiếp hay qua Steam Datagram Relay; nếu qua relay thì "ping tới game" = ping tới relay của Valve, và relay có trả lời ICMP hay không.

---

### M5 — Boss Statistics & Nightlord Predictor

**Bài toán:** Ở chế độ thường, người chơi chọn Nightlord. Ở **Deep of Night depth ≥ 3**, Nightlord bị ẩn/ngẫu nhiên → cần đoán dựa trên các boss đã gặp ở Night 1 và Night 2 (và các dấu hiệu khác).

**Thu thập dữ liệu:**
1. **Ghi nhận run** (P0): sau mỗi run người dùng nhập nhanh (hoặc chọn từ dropdown trên overlay): chế độ, depth, nhân vật, Night 1 boss, Night 2 boss, Shifting Earth (nếu có), Nightlord thực tế, kết quả.
2. **Dữ liệu prior** (P0): bảng pool boss theo Nightlord từ datamine/cộng đồng, đóng gói trong `boss_priors.json`.
3. **Dữ liệu cộng đồng** (P2): tuỳ chọn chia sẻ run ẩn danh lên server/GitHub để tăng cỡ mẫu (M6).

**Mô hình dự đoán — Naive Bayes có làm mượt:**
```
P(Nightlord = L | N1 = a, N2 = b, ctx) ∝ P(L | ctx) · P(N1 = a | L, ctx) · P(N2 = b | L, ctx, N1 = a)
```
- `ctx` = depth, map event (Shifting Earth)…
- `P(L | ctx)`: prior — đều nhau nếu không có dữ liệu.
- Likelihood: từ datamine (nếu pool boss phụ thuộc Nightlord) **kết hợp** tần suất quan sát được, dùng Laplace/Dirichlet smoothing để tránh xác suất 0 khi ít mẫu.
- Nếu boss nào **chỉ xuất hiện** với một Nightlord nhất định → loại trừ cứng (xác suất = 0 cho các Nightlord khác).

**Output:** xếp hạng Nightlord kèm %, cỡ mẫu, độ tin cậy, và gợi ý chuẩn bị (điểm yếu nguyên tố, loại relic nên dùng → link sang M3 khi M3 có).
```
Night 1: Tibia Mariner · Night 2: Demi-Human Queen
→ Gladius 46% · Fulghor 22% · Libra 15% · ...   (n = 312 run)
   Gladius yếu Holy → nên mang relic tăng sát thương Holy
```

**Thống kê:** win rate theo Nightlord/nhân vật, tần suất boss từng đêm, thời gian clear trung bình (nhập tay; tự động khi có M2).

**Weakness Cheat Sheet** (gộp vào M5 từ đề xuất cũ): bảng điểm yếu/kháng của mọi Night boss & Nightlord, hiện trên overlay khi chọn boss.

⚠️ Cần xác minh: pool boss đêm ở Deep of Night có thực sự phụ thuộc Nightlord hay hoàn toàn độc lập. Nếu độc lập, mô hình chỉ trả về prior (và app phải nói rõ "không có tín hiệu", không đưa ra % giả tạo).

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
- Lưu **bộ relic cá nhân** (local) — thay cho phần "tự tạo build" của M4 cũ, chỉ gồm relic, không có thư viện build.

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

**Liên kết M5:** khi timer chạy, run được tự điền thời gian clear và mốc Night 1/Night 2 → form ghi run của M5 chỉ còn chọn boss.

---

### M4 — Builds — *Đã bỏ*

Bỏ từ v0.3 (2026-10-10). Lý do: không tìm được nguồn build có API hoặc giấy phép mở; các nguồn đã xét (Fextralife, Game8, Reddit, YouTube) chỉ dùng được bằng cách curate tay hoặc scrape — curate tay tốn công bảo trì mỗi patch, còn scrape vi phạm ToS và dễ vỡ.

Hệ quả:
- Không có `data/builds/`, không có `internal/build`.
- Phần "lưu bộ relic cá nhân" chuyển sang M3.
- M6 không còn mục "xếp hạng build"; chỉ phục vụ M5.
- Bảng `builds` và cột `runs.build_json` đã có trong migration đầu của SQLite **giữ nguyên** (không sửa migration đã phát hành); để trống, không dùng. `build_json` có thể dùng lại để lưu bộ relic của run.

Có thể xem xét lại nếu M6 có đủ dữ liệu run (nhân vật + relic + kết quả) để tự suy ra build hiệu quả, không cần nguồn ngoài.

---

### M6 — Community Data (opt-in) — *Milestone sau*

**Mục tiêu:** Gom dữ liệu run từ nhiều người để tăng độ chính xác dự đoán Nightlord ở M5 và thống kê boss/win rate.

**Quyền riêng tư — bắt buộc:**
- **Mặc định TẮT.** Lần đầu mở app hiện màn hình giải thích rõ: gửi dữ liệu gì, để làm gì, lưu ở đâu → người dùng chọn *Chia sẻ* / *Không chia sẻ*. Đổi lại được bất cứ lúc nào trong Settings.
- Chỉ gửi **dữ liệu gameplay**: chế độ, depth, nhân vật, boss từng đêm, Nightlord, kết quả, thời gian clear, relic dùng (tuỳ chọn riêng). **Không** gửi tên Steam, IP-gắn-danh-tính, dữ liệu mạng/FPS.
- Định danh bằng **ID ngẫu nhiên** sinh trên máy (không liên kết tài khoản); có nút **"Xem dữ liệu đã gửi"** và **"Xoá dữ liệu của tôi"** trên server.
- Cấp độ chia sẻ tách riêng: ☐ Boss/Nightlord · ☐ Relic · ☐ Thời gian clear.

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

⚠️ Scope lớn → để **Phase 6**. Trong các phase trước, app vẫn lưu run **local** với schema giống hệt payload sẽ gửi lên, để khi bật M6 không phải migrate.

---

## 4. Dữ liệu game (Data Pack)

| File | Nội dung | Nguồn | Cần cho |
|------|----------|-------|---------|
| `manifest.json` | version data, game version, checksum | — | — |
| `characters.json` | nhân vật | wiki | M5, M3 |
| `bosses.json` | Night boss, Nightlord, điểm yếu, ảnh icon | datamine + wiki | M5 |
| `boss_priors.json` | pool boss theo Nightlord/depth | datamine + thống kê | M5 |
| `relic_effects.json` | hiệu ứng relic + giá trị theo tier | datamine `regulation.bin` + wiki | M3 |
| `timer_profiles.json` | thời lượng phase theo chế độ | đo thực tế + cộng đồng | M2 |

- Mọi file có JSON Schema; CI chạy `nrc-cli validate` để kiểm tra.
- App kiểm tra bản data mới (GitHub Releases), tải về, verify checksum, áp dụng không cần cập nhật `.exe`.
- Hỗ trợ đa ngôn ngữ: **Tiếng Việt + English** ngay từ đầu (tên trong data có dạng `{en, vi}`).
- Thứ tự xác minh dữ liệu đi theo roadmap: `bosses` + `boss_priors` trước, rồi `relic_effects`, cuối cùng `timer_profiles`.

---

## 5. Lộ trình (Roadmap)

| Phase | Nội dung | Kết quả |
|-------|----------|---------|
| **0 — Nền móng** ✅ | Khung Wails + Svelte, cửa sổ chính + cửa sổ overlay, `gamedata`, `store`, `config`, `ipc`, CI (lint, test, validate data), workflow release | App rỗng chạy được |
| **1 — Connection** 🟡 | ✅ M1 ping/jitter/loss tới mục tiêu cố định · ✅ FPS qua ETW (helper Admin) · ✅ overlay | Dùng được trong game |
| **2 — Boss** | M5: xác minh `bosses`/`boss_priors`, form ghi run (local), thống kê, dự đoán Bayes, weakness cheat sheet trên overlay | Dự đoán Nightlord |
| **3 — Relic** | M3: xác minh `relic_effects`, tra cứu/lọc, Relic Calculator, curse của Deep relic, lưu bộ relic cá nhân | Tra cứu relic |
| **4 — Timer** | M2: hotkey toàn cục + state machine + overlay đếm ngược + cảnh báo âm thanh; tự điền thời gian vào run của M5 | Bản 1.0 (base game) |
| **5 — Nâng cao** | M1: phát hiện kết nối game (TCP table + ETW Kernel-Network) & đo peer/relay, biểu đồ lịch sử · M2: OCR auto-sync | Bản 1.x |
| **6 — Cộng đồng** | M6 Community Data opt-in (server + aggregates) | Bản 1.x |
| **7 — DLC** | Data cho *The Forsaken Hollows* (nhân vật, Nightlord, relic, boss mới) | Milestone kế tiếp |

---

## 6. Rủi ro

| Rủi ro | Mức | Giảm thiểu |
|--------|-----|-----------|
| EAC coi app là đáng ngờ | Cao | Không inject/đọc memory; overlay là cửa sổ riêng; FPS & theo dõi mạng qua ETW; ghi rõ trong README |
| Patch game làm sai số liệu | Trung bình | Data pack tách rời, có `verified_on`, cập nhật OTA |
| Không đo được ping peer (relay/chặn ICMP) | Trung bình | Hiển thị trung thực "N/A", đo chất lượng mạng nội bộ thay thế |
| Thiếu dữ liệu cho mô hình dự đoán | Trung bình | Prior từ datamine + smoothing + hiển thị cỡ mẫu/độ tin cậy |
| Overlay không hiện trên game Fullscreen độc quyền | Thấp | Hướng dẫn dùng Borderless Windowed |
| Bản quyền hình ảnh/tên trong game | Thấp | Dự án fan, phi thương mại, credit FromSoftware/Bandai Namco |

---

## 7. Đề xuất bổ sung (chờ trao đổi)

1. **Run Tracker / Lịch sử run** — gộp dữ liệu từ M5 (boss) + M2 (thời gian) thành nhật ký mỗi run: nhân vật, kết quả, thời gian clear. Phần lớn nằm sẵn trong form ghi run của M5.
2. **Map Helper** — ghi chú vị trí theo Shifting Earth (Crater, Mountaintop, Rotted Woods, Noklateo…) và điểm đánh dấu quan trọng; có thể dùng seed map của cộng đồng.
3. **Party Mode** — 3 người cùng dùng app chia sẻ timer & dự đoán qua phòng LAN/WebSocket.
4. **Discord Rich Presence** — hiển thị Nightlord đang đánh, Day hiện tại.

---

## 8. Quyết định đã chốt & câu hỏi còn mở

### Đã chốt
| # | Ngày | Chủ đề | Quyết định |
|---|------|--------|-----------|
| D1 | 2026-10-09 | UI | Desktop GUI (Wails) **có overlay** ngay từ đầu, không làm bản CLI/TUI |
| D2 | 2026-10-09 | Timer | **Hotkey** cho bản đầu của M2; OCR auto-sync ở Phase 5 |
| D3 | 2026-10-09 | FPS | Đo FPS qua ETW là **cần thiết** → cần quyền Admin |
| D4 | 2026-10-09 | Dữ liệu cộng đồng | **Có**, gom từ nhiều người, **opt-in rõ ràng** → module M6, để Phase 6; trước đó lưu local cùng schema |
| ~~D5~~ | 2026-10-09 | Nguồn build | ~~Curate thủ công từ Fextralife / Game8 / Reddit / YouTube~~ → **thay bằng D8** |
| D6 | 2026-10-09 | DLC | **Không** trong v1 — để Phase 7 |
| D7 | 2026-10-09 | Quyền Admin | App chính chạy quyền thường; chỉ helper `--fps-helper` (cùng exe) chạy Admin qua UAC khi bật FPS |
| D8 | 2026-10-10 | Builds | **Bỏ module M4** — không có nguồn dữ liệu build dùng được |
| D9 | 2026-10-10 | Thứ tự | Sau M1: **Boss (M5) → Relic (M3) → Timer (M2)** |

### Còn mở
1. **Tên hiển thị / branding** — giữ "Nightreign Companion"? (lưu ý tránh dùng logo chính thức của game).
2. **Phát hành** — open-source trên GitHub? (có lợi cho việc cộng đồng đóng góp data pack).
3. **Phát hiện kết nối game** — có nên kéo lên sớm hơn Phase 5 không, vì hiện M1 mới đo mạng nhà chứ chưa đo kết nối thực của Nightreign?
