# Checklist kiểm chứng thủ công (lazypodman)

Tài liệu này gom **mọi thứ cần kiểm chứng trên Podman thật** từ đầu dự án tới giờ.
Mục tiêu: mỗi dòng có kết luận rõ ràng, không có chỗ nào "chắc là chạy được".

## Cách dùng

1. Chuẩn bị môi trường (mục 0).
2. Chạy từng mục, đối chiếu "Kỳ vọng".
3. Ghi kết quả vào cột **Kết quả** với đúng một trong ba nhãn:
   - `verified by human (podman <version>)` — bạn đã chạy và thấy đúng.
   - `tested against synthetic fixtures only` — chỉ có unit test với dữ liệu tự viết.
   - `not verified` — chưa chạy, chưa test.
4. Nếu sai: ghi lại output thật + version, đừng sửa code ngay trong lúc test.

Không cần làm hết trong một lần. Đánh dấu `[x]` khi xong.

## 0. Chuẩn bị

```bash
podman version                 # ghi lại version, ví dụ 5.2.3
podman info --format '{{.Host.RemoteSocket.Path}}'
systemctl --user status podman.socket 2>/dev/null || podman system service --time=0 &
```

- Linux, rootless. `podman` và socket phải **cùng store** (nếu tách store thì pods/healthcheck/kube sẽ báo off — đó cũng là một mục cần test).
- Build app: `GOFLAGS=-mod=vendor go build -o /tmp/lazypodman .`
- Terminal phụ: `watch -n1 'pgrep -a podman'` để kiểm tra không sót process.
- Version đã ghi: `podman ______`

---

## A. Pods + startup

- [ ] **A1. Tạo pod mẫu**
  ```bash
  podman pod create --name demo -p 8080:80
  podman run -d --pod demo --name web docker.io/library/nginx
  podman run -d --pod demo --name side busybox sh -c 'while :; do echo "side 12:00:00 tick: x"; sleep 2; done'
  ```
  Kỳ vọng: panel **Pods** (vị trí 3) hiện `demo`, `2 containers`, ports `8080->80/tcp`; **không** liệt kê container infra.

- [ ] **A2. Cột pod**: panel Containers có cột `demo`. Đặt `gui.showPodColumn: false`, khởi động lại → cột biến mất.

- [ ] **A3. Tabs của pod**
  - Logs: dòng của `web`/`side` xen kẽ, có prefix; dòng `side 12:00:00 tick: x` giữ nguyên.
  - Stats: tổng của pod; Containers: đúng 2 container; Config: JSON inspect.
  - Chuyển sang tab khác rồi `pgrep -a podman` → không còn process log treo.
  - Mặc định chỉ tải **200 dòng cuối**; dòng dưới cùng hiện `L: load all logs`.
  - Nhấn `L` → tải toàn bộ log của container đó, dòng dưới cùng đổi thành
    `L: load recent logs`; nhấn `L` lần nữa quay lại 200 dòng. Container khác
    vẫn giữ mặc định 200.

- [ ] **A4. Hành động**: `s` stop (có xác nhận), `S` start, `r` restart, `p` pause/unpause, `d` remove + force remove, `b` prune khi có 1 pod đã dừng.

- [ ] **A5. Phím panel**: `1`…`7` = Project/Services/Pods/Containers/Images/Volumes/Networks; dòng help dưới cùng khớp; `[`/`]` đổi tab.

- [ ] **A6. Pod không infra**: `podman pod create --name noinfra --infra=false` → pod vẫn hiện, không crash, không có marker infra.

- [ ] **A7. Chỉ có socket (không CLI)**: `CONTAINER_HOST=ssh://…` (hoặc máy chỉ mount socket) → không có panel Pods, góc dưới phải hiện `pods off: <lý do>`, Containers vẫn chạy.

- [ ] **A8. Guard version**: wrapper `podman` in ra `podman version 2.2.1` → Pods ẩn, lý do `pods off: podman too old`.

- [ ] **A9. Guard `--names`**: wrapper có `pod logs --help` **không** chứa `--names` → log gộp vẫn chạy bằng fallback từng container, không gọi `pod logs`.

- [ ] **A10. Tên pod có dấu cách**: `podman pod create --name "web api"` → thao tác đúng pod; `pgrep -a podman` thấy tên là **một** argument.

- [ ] **A11. Ports**: pod `-p 8080:80` hiện `8080->80/tcp` (lấy từ infra); pod `--infra=false` không hiện port.

- [ ] **A12. Startup chỉ 2 process**: chạy app với `pgrep -af podman` → đúng 2 process lúc khởi động (info + compose check), không có process thứ ba cho pods/kube/health.

- [ ] **A13. Fixture thật** (tuỳ chọn nhưng nên): lưu output thật vào `pkg/commands/testdata/pods/real/<version>/`:
  ```bash
  podman pod ls --format json
  podman pod inspect <pod>
  podman ps -a --format json
  podman pod logs --names <pod>
  podman logs --names <ctr>
  ```
  Rồi thêm test parse mọi file có trong thư mục đó và chạy lại `go test ./pkg/commands/`.

---

## B. Healthcheck

Tạo ba container:

```bash
podman run -d --name hc-ok  --health-cmd 'exit 0' --health-interval 2s docker.io/library/busybox sleep 600
podman run -d --name hc-bad --health-cmd 'exit 1' --health-interval 2s docker.io/library/busybox sleep 600
podman run -d --name hc-none docker.io/library/busybox sleep 600
```

- [ ] **B1. `hc-ok`**: tab **Health** liệt kê probe exit 0; bấm `t` → status line báo `healthy`, tab tự refresh.
- [ ] **B2. `hc-bad`**: các dòng exit 1 màu đỏ; `t` → **kết quả unhealthy trên status line**, KHÔNG mở error panel.
- [ ] **B3. `hc-none`**: tab ghi không có healthcheck; `t` báo tương tự và **không spawn** process.
- [ ] **B4. Container đã dừng**: stop `hc-ok`, mở lại Health → probe cũ vẫn còn; `t` → "not running", không spawn.
- [ ] **B5. Không sót process**: `pgrep -a podman` trước/sau `t` → xuất hiện 1 process, sau đó không còn.
- [ ] **B6. Chỉ socket**: `CONTAINER_HOST=ssh://…` → `t` báo `cli off: <lý do>`, không spawn; tab Health vẫn render từ inspect.
- [ ] **B7. Ghi exit code thật** (đây là mục **UNVERIFIED** quan trọng nhất):

  | Trường hợp | Lệnh | Exit code thật |
  | --- | --- | --- |
  | healthy | `podman healthcheck run hc-ok; echo $?` | ______ |
  | unhealthy | `podman healthcheck run hc-bad; echo $?` | ______ |
  | không có healthcheck | `podman healthcheck run hc-none; echo $?` | ______ |
  | container đã dừng | `podman stop hc-ok; podman healthcheck run hc-ok; echo $?` | ______ |

  Kỳ vọng hiện tại của code: `0` = healthy; `1`/`2` = unhealthy; còn lại = error panel kèm stderr nguyên văn.
  Nếu `2` không phải unhealthy, hoặc có code khác, ghi lại để sửa bảng trong design.

- [ ] **B8. Side effect**: sau `t` trên `hc-bad`, health log có thêm entry và failing streak có thể tăng; nếu container có `--health-on-failure`, hành vi đó có thể xảy ra. Ghi lại nếu thấy khác mô tả.

---

## C. Kube

- [ ] **C1. Generate**: tạo pod + container, panel Pods → `g` → Generate → preview có `kind: Pod` → lưu → `ls -l <file>` ra `-rw-------`.
- [ ] **C2. Cảnh báo plaintext**: hộp xác nhận lưu **luôn** có dòng cảnh báo env plaintext (kể cả khi pod không có env), để người dùng không bỏ qua.
- [ ] **C3. Ghi đè**: generate lần 2 → hiện prompt overwrite; Cancel giữ file cũ (mtime không đổi).
- [ ] **C4. Play**: chọn file đã lưu → output stream vào panel chính, pod xuất hiện trong Pods.
- [ ] **C5. Down**: xác nhận hiện **full path**; Cancel không spawn gì.
- [ ] **C6. Lọc file**: file `.yaml` không có `kind:` và file trong thư mục con **không** được liệt kê.
- [ ] **C7. Podman cũ**: nếu `podman kube generate/play` không có, app fallback sang `podman generate kube` / `podman play kube`; nếu cả hai không có thì hiện `kube generate off:` / `kube play off:`. Ghi lại nhánh nào đã dùng.
- [ ] **C8. Không sót process**: `pgrep -a podman` trước/sau generate → 1 process, sau đó hết.
- [ ] **C9. Chỉ socket**: cả hai tính năng kube báo off, không spawn.
- [ ] **C10. `--down` có thật không**: `podman kube play --help | grep -- --down` → ghi có/không. Nếu không có, classic down cần sửa.

---

## D. Tab Events

- [ ] **D1. Stream**: panel Containers → tab **Events** → chạy `podman run --rm busybox true` ở terminal khác → dòng sự kiện xuất hiện ngay.
- [ ] **D2. Kill khi đổi tab**: đang ở Events, chuyển sang tab khác → `pgrep -a podman` không còn `podman events`.
- [ ] **D3. Không CLI**: chỉ socket → tab ghi `cli off: <lý do>`, không spawn gì.
- [ ] **D4. Output nguyên văn**: không bị parse/mất dòng; ghi lại 2–3 dòng mẫu thật để đối chiếu sau này.

---

## E. Multi-select

- [ ] **E1. Đánh dấu**: `space` trên 2 container → mỗi dòng có `● `, status line báo `2 selected`; `space` lại → bỏ dấu, báo `selection cleared` khi hết.
- [ ] **E2. Hành động hàng loạt**: khi 2 container đang dấu, `s` → hộp xác nhận `Stop 2 containers?` → cả hai dừng; dấu được xoá sau khi thành công.
- [ ] **E3. `r` / `p` / `d`**: restart, pause/unpause, remove menu đều áp dụng cho **tất cả** mục đã dấu; `d` hiện tiêu đề `Remove N containers?`.
- [ ] **E4. Giữ dấu qua refresh**: đánh dấu 1 container rồi đợi refresh (hoặc bấm refresh) → dấu vẫn còn.
- [ ] **E5. Không dấu**: khi không đánh dấu gì, các phím trên chỉ tác động container đang chọn (hành vi cũ).

---

## F. Command palette

- [ ] **F1. Mở**: `ctrl+p` ở bất kỳ panel → menu **Command palette**.
- [ ] **F2. Nhảy panel**: chọn từng mục panel → focus đúng panel, không lỗi.
- [ ] **F3. Hành động**: các mục kube menu / healthcheck / auto-update / quit chạy đúng như phím tương ứng; `ctrl+p` trong lúc popup khác đang mở không làm treo.

---

## G. Quadlet export

- [ ] **G1. Preview + lưu**: chọn container có image/port/env/volume → `Q` → preview `.container` → xác nhận → file ở `gui.quadletDir` (mặc định `~/.config/containers/systemd`), `ls -l` ra `-rw-------`.
- [ ] **G2. Nội dung**: file có `[Container]`, `Image=`, `ContainerName=`, `Environment=` (đúng biến), `Exec=`, `PublishPort=`, `Volume=`, `Network=`, `[Service]`, `[Install]`.
- [ ] **G3. Ghi đè**: xuất lần 2 → prompt overwrite; Cancel không đổi file.
- [ ] **G4. Systemd nhận file**: `systemctl --user daemon-reload && systemctl --user start <name>` chạy được (hoặc ghi lại lỗi cụ thể). Đây là mục dễ sai nhất vì file do app tự dựng.
- [ ] **G5. Container chưa load details**: chọn container ngay sau khi mở app, bấm `Q` → app tự load details; nếu không được thì báo lỗi rõ, không crash.
- [ ] **G6. Tên xấu**: container tên `../../etc/passwd` (nếu tạo được) → file vẫn nằm trong `quadletDir`, tên đã được làm sạch.

---

## H. Auto-update

- [ ] **H1. Có trong menu**: panel Containers → `b` → có mục **Auto-update containers (podman auto-update)**.
- [ ] **H2. Xác nhận + stream**: chọn → hộp xác nhận → output `podman auto-update` stream vào panel chính → list container refresh sau khi xong.
- [ ] **H3. Không CLI**: chỉ socket → chỉ hiện thông báo `cli off: …`, không spawn.
- [ ] **H4. Không có label**: nếu không container nào có label auto-update, ghi lại output thật (kỳ vọng: không update gì, không lỗi).

---

## J. Shell tab (terminal thật)

- [ ] J1. Panel Containers, chọn container đang chạy → `x` mở menu options như cũ, trong đó có mục **exec shell** (`E`); chưa có tab shell nào trước đó.
- [ ] J2. Chọn **exec shell** → tab **shell** xuất hiện ngay **sau tab Logs** và được chọn tự động; con trỏ nằm trong vùng terminal.
- [ ] J3. Gõ `ls -la` → output đúng, màu không có nhưng layout đúng; `top`/`htop` (nếu có) vẽ được toàn màn hình và thoát bằng `q` trong app.
- [ ] J4. `vim /tmp/x` (nếu image có vim) mở đúng, gõ/xoá/di chuyển bằng mũi tên và `esc` hoạt động trong app.
- [ ] J5. `ctrl+x` → shell đóng, tab mất, quay về danh sách container; mở lại bằng `x` vẫn được.
- [ ] J6. Thoát shell bên trong (`exit`) → tab tự đóng, status line báo `shell exited`.
- [ ] J7. Resize cửa sổ terminal → nội dung co giãn đúng (thử `vim` hoặc `top`).
- [ ] J8. Container bị stop → shell tự đóng, tab biến mất, không treo app.
- [ ] J9. Chọn container khác khi đang mở shell → tab shell biến mất (shell cũ vẫn giữ); quay lại container cũ → tab xuất hiện lại, gõ tiếp được.
- [ ] J10. `E` mở terminal trong tab shell; `x` vẫn mở options menu như cũ; `a` vẫn attach native.

## I. Regression chung

- [ ] **I1. Không có popup cho healthcheck**: kết quả healthy/unhealthy chỉ ở status line.
- [ ] **I2. Thông báo off chỉ hiện sau khi probe thất bại thật**: lần đầu mở app, chưa bấm `t`/`g` → không có dòng `health run off:` / `kube … off:`.
- [ ] **I3. Status line 10 giây**: flash biến mất sau ~10s, không đè lên waiting status.
- [ ] **I4. `t`/`g`/`Q`/`space`/`ctrl+p` không đụng phím cũ** (xem `docs/keybindings/Keybindings_en.md`).
- [ ] **I5. `--version` và `-c`**: chạy được, `-c` in `quadletDir: ""` và `kubeDir: ""`, không cảnh báo.

---

## Bảng tổng hợp kết quả

| Mục | Nội dung | Kết quả | Ghi chú (version / output) |
| --- | --- | --- | --- |
| A1–A5 | Pods cơ bản + tabs + phím | | |
| A6–A11 | Pods edge case | | |
| A12 | Startup 2 process | | |
| A13 | Fixture thật | | |
| B1–B6 | Healthcheck UI + key | | |
| B7 | Bảng exit code | | |
| B8 | Side effect health log | | |
| C1–C9 | Kube generate/play/down | | |
| C10 | `kube play --down` tồn tại | | |
| D1–D4 | Events tab | | |
| E1–E5 | Multi-select | | |
| F1–F3 | Palette | | |
| G1–G6 | Quadlet | | |
| H1–H4 | Auto-update | | |
| I1–I5 | Regression | | |

Khi xong, các chỗ đang ghi "chưa kiểm chứng" trong
`docs/superpowers/specs/2026-11-18-healthcheck-design.md` và
`docs/superpowers/plans/2026-11-18-*.md` sẽ được cập nhật theo bảng này.

## Tài liệu liên quan

- Design + plan từng phase: `docs/superpowers/specs/`, `docs/superpowers/plans/`
  - Pods: `2026-11-18-pods-design.md`, `2026-11-18-pods.md`
  - Healthcheck: `2026-11-18-healthcheck-design.md`, `2026-11-18-healthcheck.md`
  - Kube: `2026-11-18-kube-design.md`, `2026-11-18-kube.md`
  - UX (Events / multi-select / palette): `2026-11-18-ux-design.md`, `2026-11-18-ux.md`
  - Quadlet + auto-update: `2026-11-18-quadlet-design.md`, `2026-11-18-quadlet.md`
- Bảng "Verification status" cuối mỗi design là nơi cần cập nhật sau khi bạn chạy xong.

---

## Kết quả đã chạy (podman 6.2.0, sandbox root + vfs, không systemd/không runtime)

Môi trường: `podman version 6.2.0`, chạy bằng `--storage-driver=vfs` trong
`/tmp/podman-root`, API qua `unix:///run/podman/podman.sock`. Container **create/inspect
được** nhưng **start bị chặn** (`cgroupfs` read-only, không `/dev/net/tun`), nên mọi mục
cần container chạy vẫn chưa kiểm chứng.

Đã chạy tự động (`-tags integration`):

| Mục | Kết quả | Ghi chú |
| --- | --- | --- |
| Startup check (số process, thời gian) | **verified by agent (podman 6.2.0)** | 1 process, **185 ms**; test fail nếu > 3 s |
| Parse `pod pod ls` + `ps -a` JSON | **verified by agent (podman 6.2.0)** | danh sách rỗng vẫn parse đúng; container tạo bằng CLI được map |
| Socket API + `RefreshPodState` | **verified by agent (podman 6.2.0)** | socket list thấy container vừa tạo, pod snapshot không lỗi |
| C7: chọn cú pháp kube | **verified (6.2.0)** | prefix chọn `kube generate` / `kube play` |
| C10: `podman kube play --down` | **verified (6.2.0)** | flag tồn tại |
| KubeGenerate với container thật | **verified (6.2.0)** | output có `kind: Pod`, `Created with podman-6.2.0` |
| D1/D4: events stream | **verified (6.2.0)** | nhận dòng thật: `... container create ... (image=..., name=...)` |
| G2: unit Quadlet từ inspect thật | **verified (6.2.0)** | có `[Container]`, `Image=`, `ContainerName=`, `PublishPort=`, `Volume=`, env sorted |
| G4: systemd nhận file | **không kiểm chứng được ở đây** | `podman quadlet install` cần systemd dbus; sandbox không có. Cần máy có `systemctl --user` |
| B1–B8 healthcheck, A1–A11 pods, E/F UI, H auto-update | **not verified** | cần container chạy và TUI thật |

Lệnh tái lập trong repo:

```sh
CONTAINERS_STORAGE_CONF=/tmp/storage.conf \
GOFLAGS=-mod=vendor go test -tags integration ./pkg/commands/ -run Integration -v
```
