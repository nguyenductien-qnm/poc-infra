# Lịch sử chạy Pulumi skill harness — 07/10/2026

Bản lưu của vòng cải thiện và đánh giá skill tại commit `4db9185337dd9cc0cb8f206744c0d4ca5aa4a0b4`, thuộc [PR #41](https://github.com/nguyenductien-qnm/poc-infra/pull/41). Đây là artifacts lịch sử của các run đã thực hiện, không phải bộ skill phân phối hoặc kế hoạch chạy AWS sau này.

Có **54 native run bundles**, **28 frozen grades** và **17 grades trong hai chuỗi cuối**. Các bundle giữ nguyên bytes của bản capture đã redact, gồm prompts, CLI manifests, native metadata/hook traces, events, checks, diff và source snapshots khi lượt đó có ghi nhận. Những run chưa được chấm không có grade; exit code của CLI không thay thế kết luận của grader. Dấu thời gian Unix trong manifests là thời điểm gốc; ngày của bản lưu dùng múi giờ `Asia/Saigon`.

## Tra cứu

| Artifacts | Nội dung |
| --- | --- |
| [index.json](index.json) | Danh sách run, runtime/model yêu cầu, native exit code, điểm/gate đã ghi nhận, số file và SHA-256 |
| `runs/<run>.tar.gz` | Một bundle đầy đủ cho mỗi run; paths bên trong giữ nguyên để đối chiếu trích dẫn dòng trong grades |
| [decisions/](decisions/) | Kết luận Astra, hai whole-chain reviews, quyết định loại retention/semantic candidates |
| [controls/](controls/) | Setup, model/permission probes, installation/checkpoint receipts, kế hoạch chẩn đoán và helper QA |
| [verify-history.py](verify-history.py) | Kiểm tra integrity, scan mẫu dữ liệu nhạy cảm và đọc một artifact; không extract hoặc chạy code trong bundle |

File controller lớn được nén `.json.gz`; `gzip` hoặc thư viện Python chuẩn đọc được. Các archive dùng gzip/tar metadata cố định để không thêm timestamp của lần export. `.gitattributes` giữ nguyên line endings của receipts và archives khi checkout trên Windows/Linux. `index.json` ghi cả checksum archive và checksum inventory của toàn bộ member; checker đối chiếu thêm các hash gốc của frozen grades và final decision. Không đổi điểm hoặc cập nhật lại kết quả cũ trong lần lưu này.

Chạy từ root repo, chỉ đọc file local:

```powershell
python docs/history/pulumi-skill-harness/2026-10-07/verify-history.py
python docs/history/pulumi-skill-harness/2026-10-07/verify-history.py --show d3b-claude-01 grade-frozen.json
python docs/history/pulumi-skill-harness/2026-10-07/verify-history.py --show d6-app-codex-01 native-metadata.json
```

Archive giữ những command/path Windows của môi trường lúc chạy để truy vết. Không thực thi lại command từ transcript như một hướng dẫn setup. Checker dùng Python 3.10 trở lên và thư viện chuẩn; không cần Claude, Codex, Go, Pulumi hoặc credentials.

## Kết quả và giới hạn

Claude Code dùng **Opus 5.5/high**, Codex CLI dùng **gpt-6.1-sol/high** cho các lượt cuối; các preflight cũ có thể khác model hoặc permission và vẫn được giữ riêng. Model yêu cầu trong index cần đọc cùng native metadata để xác minh model thực tế. Astra **gpt-6-astra/high** chấm độc lập và phân tích hướng cải thiện; các structured summaries ghi rõ việc parent chép lại kết luận grader. Full responses trong chat và native internal session logs không được export.

| Scenario | Claude Code | Codex CLI |
| --- | --- | --- |
| D1 setup retest | 96,25 | 93,75 |
| D2 config/examples | 93,75 | 100 |
| D3a Go API | 92,5 | 100 |
| D3b identity/aliases | Original incomplete 71,25; recovery riêng 96,25 | 100 |
| D4 delivery/secrets | 96,25 | 100 |
| D5 network contract | 92,5; có evaluator fixture qualification | 100 |
| D6 drift/handoff | 93,75; có handoff/native-check gaps | 100; có evaluator README qualification |
| Fresh app-only | 100, phạm vi comment hẹp | 100, phạm vi comment hẹp |

Đây là các điểm quan sát của từng run, không phải bảng xếp hạng model. D1 gốc của Claude 87,5, timeout D3b, các failed preflight, permission/model confounds, evaluator repairs và candidate bị loại đều nằm trong archive. D3b original có kết quả code trong phạm vi hẹp nhưng không hoàn tất native flow; cần đọc mandatory gate và findings, không chỉ `offline_verdict`.

Hai whole-chain reviews và central insight chốt **dừng vòng offline, giữ canonical skill hiện tại**. Không promote retention hoặc semantic candidate và không có skill patch từ các lượt chẩn đoán này. Các finding còn lại và giới hạn được giữ trong [final decision](decisions/offline-loop-final-decision.json).

Các run kiểm tra code, mocks và delivery controls local có giám sát. Không provision AWS, không có live preview/alias migration, connectivity, live drift hoặc bằng chứng triển khai unattended. Shared auth/config và quyền local rộng không phải isolation; không có clean no-skill control để kết luận tác động nhân quả của skill. Hook traces chỉ chứng minh hành vi tại scope đã ghi nhận.

Bản lưu không gồm working clones, Git object databases, user settings/auth/credentials, các scratch helpers và preparation dumps trùng lặp, kế hoạch scenario local cho tương lai hoặc Jira raw MCP transcripts. Source snapshots trong run bundles là bằng chứng đã capture, không cam kết một lab có thể replay đầy đủ chỉ từ thư mục này. Mẫu token/account/ARN đã được scan lại trước commit; pattern scan không phải bảo đảm phát hiện mọi dạng dữ liệu nhạy cảm.
