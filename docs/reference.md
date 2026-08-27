# 参考资料

与本项目相关的工具与开源项目(数据来源本身见 [data.md](data.md):
原始 pak → `~/Downloads/rocom/Paks/`,解包产物 → `~/Downloads/rocom/parsed/`)。

## 解包

| 项目 | 说明 |
| --- | --- |
| [CUE4Parse](https://github.com/FabianFG/CUE4Parse) | `scripts/unpack` 的解析引擎,内置 `GAME_RocoKingdomWorld` 游戏支持(自定义 AES 变体/Bin/luac,无需 usmap);其 `FRocoBinData.cs` 也是 `scripts/bin2json.py` 解 `.bytes` 配置的算法参照。构建需本地克隆,位置用环境变量 `CUE4PARSE_DIR` 指定(默认 `~/Git/gh/CUE4Parse`) |
| [FModel](https://github.com/4sval/FModel) | Windows GUI 解包器,手动导出备用路径,可与 `scripts/unpack.sh` 互为校验(该游戏条目 UeVersion=68812827 即 `GAME_RocoKingdomWorld`) |

## 协议与同类项目

| 项目 | 说明 |
| --- | --- |
| [lsj9383/blog](https://github.com/lsj9383/blog) | tsf4g 通信协议说明 |
| [h3110w0r1d-y/rocom-helper](https://github.com/h3110w0r1d-y/rocom-helper) | 闭源洛克王国世界助手,本项目受其启发 |
| [yuzeis/Roco-Kingdom-Protocol-Parser](https://github.com/yuzeis/Roco-Kingdom-Protocol-Parser) | 开源洛克王国协议解析器,简称 RKPP |

## 姊妹项目(同一套解包数据衍生,构建与数据流程互不依赖)

| 项目 | 说明 |
| --- | --- |
| [rocom-pets](https://github.com/whoisnian/rocom-pets) | 桌宠:宠物模型/动画/材质还原(Rust 运行时 + C# 导出器)。shader 逆向与 3D 渲染的全部工具与文档在那边 |
| [rocom-petvo](https://github.com/whoisnian/rocom-petvo) | 宠物叫声图鉴。解包侧的 bnk/wem 关联链路记在本仓库 [audio.md](audio.md) |

三边共用 `scripts/unpack.sh` 这个解包入口,但各自的生成物、构建与运行都不互相依赖。
**唯一的一条运行期联系**是宠物详情页那张炫彩色卡上的链接:点它跳 rocom-pets 的站点
[rkpet.whoisnian.com](https://rkpet.whoisnian.com),看同一只、同一形态、同一套炫彩的 3D 效果。

走的是那边给外部工具开的接口(它的 `web/README.md` 有完整说明):

    GET https://rkpet.whoisnian.com/api/link?petbase=<形态编号>&shiny=1&glass=<type>:<value>

送进去的全是**游戏自己的编号** —— 形态编号是 `PetData.base_conf_id`,`glass` 是
`GlassInfo{glass_type, glass_value}` 原样,异色是 `mutation_type & 1`。「形态编号 → 它那边的
包名/资产名」和它 `look` 参数的写法都由它自己换算,本仓库不抄:那两样随它的版本走,
抄过来就是两处要同步。默认回 302,所以前端就是个普通 `<a href>`,不点不会有任何外部请求
(本项目是局域网工具,断网照常用)。

## 已弃用(仅留作历史对照)

| 项目 | 说明 |
| --- | --- |
| [phainia/pak-public-kit](https://github.com/phainia/pak-public-kit) | 曾为名称表源;其 PET_CONF 本地化整体错位(见 data.md 第 5 节),已被自有解包替代 |
| [kikozz/Roco-Kingdom-World-Data-2026-05-21](https://github.com/kikozz/Roco-Kingdom-World-Data-2026-05-21) | 曾为 `.proto` 源;现字段号直接取自解包描述符 all.pb,其 Bin JSON 可作名称表三方对照 |
