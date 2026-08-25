import React from 'react'

// ContextMenu 列表项的右键/长按菜单(fixed 定位在指针处;打开/关闭逻辑在父级)。
export default function ContextMenu({ menu, onDetail, onCopy, onFilterSame }) {
  if (!menu) return null
  const eggGroups = (menu.pet.eggGroups || []).map((g) => g.name)
  return (
    <div className="ctx-menu" style={{ left: menu.x, top: menu.y }} onClick={(e) => e.stopPropagation()}>
      <div className="ctx-item" onClick={() => onDetail(menu.gid)}>查看详情</div>
      <div className="ctx-item" onClick={() => onCopy(menu.gid)}>复制编号</div>
      <div className="ctx-sep" />
      <div className="ctx-item" onClick={() => onFilterSame({ search: menu.pet.species })}>筛选相同种类</div>
      <div className="ctx-item" onClick={() => onFilterSame({ nature: menu.pet.nature, natureExclude: '' })}>筛选相同性格</div>
      <div className="ctx-item" onClick={() => onFilterSame({ speciality: menu.pet.speciality })}>筛选相同特长</div>
      {eggGroups.length > 0 && (
        // 配对候选:蛋组多选是「或」,双蛋组(如妖精+巨灵)的宠物筛出属于其中任一组的全部宠物。
        <div className="ctx-item" title={'蛋组 ' + eggGroups.join(' / ') + ' 任一'}
          onClick={() => onFilterSame({ eggGroup: eggGroups })}>筛选配对蛋组</div>
      )}
    </div>
  )
}
