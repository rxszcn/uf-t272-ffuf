# T272 交付说明：同关键词多词表静默塌缩的定位与修复

## 1. 根因

两条链路叠加造成静默漏测：

1. **装配层**（`pkg/ffuf/optionsparser.go` `ConfigFromOptions`）：`-w` 未带 `:KEYWORD` 时一律回填默认关键词 `FUZZ`，`--input-cmd` 同理。两条都不带关键词的词表因此得到相同关键词，装配时不做任何查重。
2. **输入值模型**（`pkg/input/input.go`）：`clusterbombValue()` / `pitchforkValue()` 把每轮载荷写入 `map[string][]byte`，键为 `p.Keyword()`。同关键词的后挂词表在 map 中覆盖先挂词表，而 `Total()` 仍按各 provider 条目数做笛卡尔积计数。

结果：报告的组合完成数 = 两表行数之积（如 3×2=6），进度与结果行数自洽，但实际发出的每个请求只携带后挂词表的载荷，先挂词表的整维载荷从未出现在任何请求里，且每个实际发出的载荷按先挂词表行数重复多轮。操作员无从察觉，漏测组合中的未授权访问路径 / 弱口令凭据必然漏报。显式不同关键词的对照组走的是不同的 map 键，行为完全正常——与题面观测一致。

同源问题还有两处静默放行：

- **空关键词**：`词表路径:`（尾随冒号）解析出 `Keyword == ""`，空字符串同样是合法 map 键，可与另一条空关键词词表互相塌缩，且 `keywordPresent` 用 `strings.Contains(x, "")` 恒真，过滤层也拦不住。
- **外部命令输入源**：`--input-cmd` 与 `-w` 共用同一默认关键词与同一值模型，同关键词的「词表 + 命令」组合以同样方式塌缩。

## 2. 处置方案与语义边界

**方案：装配期硬失败（fail-fast），不做自动改名、不做静默合并。**

- `ConfigFromOptions` 在全部 InputProvider（词表 + 外部命令）装配完成后统一校验：关键词为空 → 报错；关键词重复 → 报错并指明冲突双方的路径 / 命令。错误进入既有 multierror，主流程按既有错误路径终止，扫描不启动、零请求发出。
- `MainInputProvider.AddProvider`（`pkg/input`）做同构校验：空关键词或与前序 provider 重复的关键词直接返回 error。这覆盖绕过选项解析的装配方——`main.go` 的结果复现打印与 `pkg/assembly/build.go` 的任务装配（含跨任务复跑的历史重放）都走 `NewInputProvider` 这同一路径，重放遇到坏配置同样响亮失败，重放结论不会带着塌缩的组合数进入复盘底稿。

**语义边界与理由：**

- **空关键词判错而非回填 FUZZ**：尾随冒号几乎必是笔误。若静默回填 FUZZ，与另一条默认词表并存时会重新制造本工单要消灭的塌缩场景；显式报错迫使操作员在「去掉冒号用默认 FUZZ」与「写清关键词」之间做显式选择。错误信息中已写明这两种改法。
- **同关键词判错而非自动派生唯一关键词**：自动改名（如 FUZZ2）会让请求模板里的 `FUZZ` 占位符只对应其中一条词表，另一条词表的载荷无处可去——塌缩从「静默丢载荷」变成「静默发不出」，本质未变。组合扫描的语义前提就是「一维载荷一个可寻址关键词」，配置不满足前提时唯一正确的处置是拒绝启动。
- **显式不同关键词的既有行为逐项不动**：校验只在「空」或「重复」时触发，不触碰解析、编码器挂接、关键词过滤与三种迭代模式的任何既有路径。

## 3. 改动面（按文件逐一定位）

| 文件 | 位置 | 改动 |
|---|---|---|
| `pkg/ffuf/optionsparser.go` | `ConfigFromOptions`，词表 / 命令装配循环之后、`len(conf.InputProviders) == 0` 检查之前（约第 395–413 行） | 新增统一校验循环：空关键词报错（提示去冒号或显式命名）；重复关键词报错并列出冲突双方来源。词表与外部命令输入源一并收敛 |
| `pkg/input/input.go` | `MainInputProvider.AddProvider`（约第 42–57 行） | 装配入口前置校验：空关键词、与已挂 provider 重复的关键词均返回 error，经 `NewInputProvider` 汇入 multierror |
| `pkg/ffuf/optionsparser_test.go` | 文件尾部新增 4 个用例 | 同默认关键词双词表拒绝、词表+命令同关键词拒绝、尾随冒号空关键词（词表与命令两路）拒绝、显式不同关键词对照组正常装配 |
| `pkg/input/wordlist_test.go` | 文件尾部新增 3 个用例 | `NewInputProvider` 层：同关键词拒绝、空关键词拒绝、显式不同关键词 clusterbomb 全笛卡尔积对照（6 组合各出现恰一次、先挂词表 3 个载荷全部出现） |

## 4. 题面要求与改动对应关系

| 题面要求 | 对应改动 |
|---|---|
| 同关键词多词表不得再静默塌缩 | optionsparser.go 重复关键词校验 + input.go `AddProvider` 查重，两处报错 |
| 报告组合完成数与实际请求集合一致 | 塌缩配置在装配期终止，扫描不启动；不存在「计数 6、实发 3」的报告 |
| 显式不同关键词组合行为逐项不回归 | 校验不触碰正常路径；两个对照测试 + 实测运行 3 锁定 3×2=6 请求、6 条不同路径、每组合恰一次 |
| 尾随冒号空关键词不得静默放行 | optionsparser.go 空关键词校验（词表、命令两路）+ input.go 同构校验 |
| 外部命令输入源一并收敛 | 校验位于词表与命令装配循环之后，对两类 provider 统一生效；专设词表+命令同关键词测试 |
| 历史重放走同一装配路径 | `pkg/assembly/build.go` → `NewInputProvider` → `AddProvider`，重放命中坏配置同样失败 |
| 改动面收在 pkg/ffuf 与 pkg/input 及其测试 | 仅上表 4 个文件，新增断言落位于各自既有测试文件，沿用既有测试框架 |

## 5. 验收命令与实测读数

### 5.1 测试套件

```
go test ./...
```

读数：全部 12 个包 `ok`，零失败（含 `pkg/ffuf`、`pkg/input` 新增 7 个断言用例，既有测试全部保持通过）。

### 5.2 构造输入实测（快照环境、零外部依赖）

构造输入：`w1.txt`（alpha/bravo/charlie 三载荷）、`w2.txt`（delta/echo 两载荷）；本地 `127.0.0.1:18231` 日志服务器逐请求记录路径。

**运行 1（同默认关键词，触发处置）**

```
ffuf.exe -u http://127.0.0.1:18231/FUZZ -w w1.txt -w w2.txt -mc all
```

读数：退出码 1；报错 `Keyword "FUZZ" is used by multiple input providers ("…w1.txt" and "…w2.txt"). Each wordlist / input command needs a unique keyword, eg. -w '/path/to/wordlist:KEYWORD'`；服务器请求日志 0 条扫描请求——处置语义为「装配期拒绝、扫描不启动」，与第 2 节一致。

**运行 2（尾随冒号空关键词，触发处置）**

```
ffuf.exe -u http://127.0.0.1:18231/FUZZ -w w1.txt: -mc all
```

读数：退出码 1；报错 `Input provider "…w1.txt" has an empty keyword. Remove the trailing colon to use the default FUZZ keyword, or specify a non-empty keyword`。

**运行 3（显式不同关键词对照组）**

```
ffuf.exe -u http://127.0.0.1:18231/W1/W2 -w w1.txt:W1 -w w2.txt:W2 -mc all -s
```

读数：退出码 0；结果行 6 行；服务器请求日志 6 条扫描请求、6 条不同路径（`/alpha/delta`、`/alpha/echo`、`/bravo/delta`、`/bravo/echo`、`/charlie/delta`、`/charlie/echo`），每条恰出现一次——请求数 = 不同路径数 = 3×2 笛卡尔积，先挂词表 3 个载荷全部实发。
