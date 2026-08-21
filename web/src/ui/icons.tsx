import type { SVGProps } from 'react'

type IconProps = SVGProps<SVGSVGElement>

function Icon({ children, ...props }: IconProps) {
  return (
    <svg aria-hidden="true" fill="none" stroke="currentColor" viewBox="0 0 24 24" {...props}>
      {children}
    </svg>
  )
}

export const CalendarIcon = (props: IconProps) => (
  <Icon {...props}><rect x="4" y="5" width="16" height="15" rx="2" /><path d="M8 3v4m8-4v4M4 10h16" /></Icon>
)
export const BagIcon = (props: IconProps) => (
  <Icon {...props}><path d="m5 8 2 12h10l2-12M8 8V6a4 4 0 0 1 8 0v2" /></Icon>
)
export const PlusIcon = (props: IconProps) => <Icon {...props}><path d="M12 5v14M5 12h14" /></Icon>
export const SwapIcon = (props: IconProps) => (
  <Icon {...props}><path d="M20 7h-9a5 5 0 0 0-5 5m-2 5h9a5 5 0 0 0 5-5M17 4l3 3-3 3M7 14l-3 3 3 3" /></Icon>
)
export const TrashIcon = (props: IconProps) => <Icon {...props}><path d="M5 7h14m-9-3h4m-7 3 1 13h8l1-13" /></Icon>
export const ArrowIcon = (props: IconProps) => <Icon {...props}><path d="m9 18 6-6-6-6" /></Icon>
export const CloseIcon = (props: IconProps) => <Icon {...props}><path d="m6 6 12 12M18 6 6 18" /></Icon>
export const RecipeIcon = (props: IconProps) => <Icon {...props}><path d="M6 3h12v18H6zM9 8h6m-6 4h6m-6 4h4" /></Icon>
