import React from 'react'
import { IconsContext } from '../context'
import { imgURL, useImgFallback, InlineIcon } from './icons'

// 宠物名称行内的各种小徽标(性别/异色炫彩/血脉/形态/蛋组/系别/搭档标记)。

// Gender 渲染性别符号(♂ 蓝、♀ 粉)。**自己画,不用 ♂/♀ 字符**:这两个码位在各系统落到
// 不同的符号字体,有的当数学符号画(居中于数学轴,比正文高出一截)、有的走彩色 emoji,
// 与相邻的名字/等级对不齐(iOS 上尤其明显)。SVG 的框由 CSS 定死,各浏览器摆在同一处。
export function Gender({ g }) {
  if (g !== '♂' && g !== '♀') return null
  const male = g === '♂'
  return (
    <svg className={'gender ' + (male ? 'male' : 'female')} viewBox="0 0 16 16"
      role="img" aria-label={male ? '雄性' : '雌性'}
      fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      {male
        ? <><circle cx="6.5" cy="9.5" r="4" /><path d="M9.4 6.6 14 2" /><path d="M9.5 2H14v4.5" /></>
        : <><circle cx="8" cy="6" r="4" /><path d="M8 10v5M5.5 12.8h5" /></>}
    </svg>
  )
}

// Form 渲染地区/季节形态徽标(普通宠物为空)。
export function Form({ form }) {
  if (!form) return null
  return <span className="mark mark-form" title="形态">{form}</span>
}

// MarkIcon 渲染单个异色/炫彩标记图;无图或加载失败退化为原文字徽标(异/彩)。
function MarkIcon({ src, title, fallback, cls }) {
  const [bad, onError] = useImgFallback(src)
  if (src && !bad) {
    return <img className="mark-img" src={imgURL(src)} alt={title} title={title} onError={onError} />
  }
  return <span className={'mark ' + cls} title={title}>{fallback}</span>
}

// Marks 渲染异色/炫彩标记(优先游戏图标;两者兼具用合成的异色炫彩图)。
// 炫彩优先用「这一种炫彩」自己的标记图(隐藏炫彩每季一张,异色时后端已换成异色炫彩合成版),
// 故炫彩分支只出一枚图,不再另加异色标;后端查不到这一款(新赛季款)时退回通用图标。
export function Marks({ p }) {
  const icons = React.useContext(IconsContext)
  if (!p) return null
  if (p.colorful) {
    const src = (p.glass && p.glass.icon) || (p.shiny ? icons.shinyColorful : icons.colorful)
    const kind = p.shiny ? '异色炫彩' : '炫彩'
    return <MarkIcon src={src} title={p.glass ? `${kind} · ${glassDesc(p.glass)}` : kind}
      fallback={p.shiny ? '异彩' : '彩'} cls="mark-colorful" />
  }
  return p.shiny ? <MarkIcon src={icons.shiny} title="异色" fallback="异" cls="mark-shiny" /> : null
}

// glassDesc 一行外观描述:主名在前、补一句限定语 ——
// 普通炫彩「配色 粒子」(亮X亮 - 紫橙 四角星),隐藏炫彩「外观名 赛季归属」(暗夜拾光 第1赛季限定)。
// 详情页的色卡不再重复这段(它只说点了跳哪儿),故这里是唯一出处。
// 与后端 GlassDesc(实时地图的野生宠与花种用)同序,只是那边接在一起写「配色·粒子」——
// 这条整句已经被 ` · ` 断过一次,再用一个点会看不清哪层是哪层。
export function glassDesc(g) {
  const tail = g.hidden ? g.season : g.particle
  return tail ? `${g.name} ${tail}` : g.name
}

// Blood 渲染血脉(主图标 + 中文短名);iconOnly=仅图标(列表用,名称落到 title)。
export function Blood({ p, iconOnly }) {
  if (!p || !p.blood) return null
  return (
    <span className="blood" title={'血脉 ' + p.blood}>
      <InlineIcon src={p.bloodIcon} className="blood-ic" alt={p.blood} />{!iconOnly && p.blood}
    </span>
  )
}

// EggGroups 展示宠物蛋组(繁殖组)标签,每个组名 hover 显示官方描述;无蛋组返回 null。
export function EggGroups({ groups }) {
  if (!groups || !groups.length) return null
  return (
    <span className="egg-groups">
      {groups.map((g) => (
        <span key={g.id} className="egg-group" title={g.desc ? `蛋组 · ${g.desc}` : '蛋组'}>{g.name}</span>
      ))}
    </span>
  )
}

// Types 渲染系别(icons 与 types 一一对应,前置属性小图);plain=去掉色块背景,仅图标+文字。
export function Types({ types, icons, plain }) {
  const list = types || []
  const cls = plain ? 'type type-plain' : 'type'
  return (
    <>
      {list.map((t, i) => (
        <span key={i} className={cls} data-t={t}>
          <InlineIcon src={icons && icons[i]} className="type-ic" alt="" />{t}
        </span>
      ))}
      {list.length === 0 && <span className="muted">-</span>}
    </>
  )
}

// PetMark 渲染搭档标记徽章(橙色外框底 img_collect + 白色标记符号),叠在头像左上角;
// 无标记(值 0=无)或缺符号图时不渲染。
export function PetMark({ p }) {
  const icons = React.useContext(IconsContext)
  if (!p || !p.partnerMarkIcon || p.partnerMark === '无') return null
  return (
    <span className="pet-mark" title={p.partnerMark}>
      {icons.partnerFrame && <img className="pet-mark-frame" src={imgURL(icons.partnerFrame)} alt="" />}
      <img className="pet-mark-ic" src={imgURL(p.partnerMarkIcon)} alt={p.partnerMark} />
    </span>
  )
}
