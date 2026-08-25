import { useState, useEffect } from 'react'
import { getFlowers, subscribe } from '../../api'

// —— 稀兽花种图层(见 docs/map.md 8)——
// 大世界同时开着二十来朵花种,每朵关着一只混血精灵。点位不是固定的:普通花种每天凌晨 4 点、
// 命定花种每两周周五凌晨 4 点在候选点里重投一遍,战斗里被捕捉/击败的那朵也当场重投,
// 故整层由后端从流量里的花种列表推过来(见 internal/pipeline/flowers.go),前端只管开关与摆放。
//
// **参观好友世界时画的是好友的花**(方便提醒他去捉):后端据 0x039d 判断在谁的世界里,
// 载荷的 visit 标志说明这一套是不是别人的,图层名随之标出来——否则分不清图上这些是谁的花。
//
// **三态,不是两态**:花种列表里没有炫彩,炫彩只在玩家点开某朵花时服务器才单独下发。
// 被动抓包不能代替客户端去问,所以没点过的花永远是「未检测」——绝不能显示成「普通」。
const LS_KEY = 'map.flowerLayer'     // 图层开关(默认关)
const LS_DETECT = 'map.flowerDetect' // 检测模式开关(默认关)
const load = (key) => { try { return localStorage.getItem(key) === '1' } catch { return false } }
const save = (key, v) => { try { localStorage.setItem(key, v ? '1' : '0') } catch { /* 隐私模式等 */ } }

// 检测状态(与后端 store.Flower* 对应)。
export const FL_UNDETECTED = 0 // 还没为这朵花收到过详情
export const FL_PLAIN = 1      // 检测过:非炫彩
export const FL_GLASSY = 2     // 检测过:炫彩

const ST_NAME = { [FL_UNDETECTED]: '未检测', [FL_PLAIN]: '普通', [FL_GLASSY]: '炫彩' }

// flowerTitle 组一条花种标记的悬停说明:`火神 Lv.60 炫彩` / `铠甲虫 Lv.55 未检测`。
// 等级后端按星级查表算好(花种列表一到就有,不必等检测);种族在花里的精灵被采收后会变成未知,
// 那时退成「稀兽花种 Lv.55 未检测」。
export function flowerTitle(f) {
  const parts = [f.n || '稀兽花种']
  if (f.lv) parts.push('Lv.' + f.lv)
  parts.push(ST_NAME[f.st] || ST_NAME[FL_UNDETECTED])
  return parts.join(' ')
}

// useFlowers 管理花种图层:订阅后端推送、按当前场景与开关筛出可绘制的标记。
export function useFlowers(account, res) {
  const [flowers, setFlowers] = useState([])
  const [visit, setVisit] = useState(false) // 当前这套花是不是好友世界的
  const [on, setOn] = useState(() => load(LS_KEY))
  const [detect, setDetect] = useState(() => load(LS_DETECT))

  useEffect(() => {
    let alive = true
    setFlowers([])
    getFlowers().then((d) => {
      if (!alive || !d) return
      setFlowers(d.flowers || [])
      setVisit(!!d.visit)
    }).catch(() => {})
    return () => { alive = false }
  }, [account])

  // 后端每次列表刷新/检测出结果/花被打掉都推全量,直接替换即可(都是低频事件)。
  useEffect(() => subscribe((m) => {
    if (m.type === 'flowers') {
      setFlowers(m.data.flowers || [])
      setVisit(!!m.data.visit)
    }
  }), [account])

  const toggle = () => setOn((v) => { save(LS_KEY, !v); return !v })
  const toggleDetect = () => setDetect((v) => { save(LS_DETECT, !v); return !v })

  // 花种只在有底图的大世界场景刷,进副本/家园时本层自然为空。
  const here = flowers.filter((f) => f.res === res)
  // 检测模式:只隐藏**已确认不是炫彩**的那些,未检测的照常显示(方向永远安全,
  // 与眠枭之星收集模式同一原则:宁可多显示,不能藏掉可能是炫彩的)。
  const marks = on ? here.filter((f) => !detect || f.st !== FL_PLAIN) : []
  const glassy = here.filter((f) => f.st === FL_GLASSY).length

  return { marks, num: here.length, glassy, visit, on, toggle, detect, toggleDetect }
}
