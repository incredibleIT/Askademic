import feishuIcon from '@/assets/img/datasource-feishu.ico'
import larkIcon from '@/assets/img/datasource-lark.svg'
import dingtalkIcon from '@/assets/img/im/dingtalk.svg'

export const datasourceIconMap: Record<string, string> = {
  feishu: feishuIcon,
  lark: larkIcon,
  // Drive (云盘) connectors reuse the wiki icons - same product, same brand.
  feishu_drive: feishuIcon,
  lark_drive: larkIcon,
  dingtalk: dingtalkIcon,
}

export function getDatasourceIconUrl(type: string): string | undefined {
  return datasourceIconMap[type]
}
