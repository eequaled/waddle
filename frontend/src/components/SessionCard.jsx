import { Chrome, Code, Slack, MessageSquare, Music, FileText, Monitor } from 'lucide-react'
import { cn } from '../lib/utils'

const AppIcon = ({ app, className }) => {
    switch (app.toLowerCase()) {
        case 'chrome': return <Chrome className={className} />
        case 'code': return <Code className={className} />
        case 'slack': return <Slack className={className} />
        case 'spotify': return <Music className={className} />
        case 'notes': return <FileText className={className} />
        default: return <Monitor className={className} />
    }
}

export function SessionCard({ title, tags, time, apps, isActive, onClick }) {
    return (
        <div
            onClick={onClick}
            className={cn(
                "p-4 rounded-xl border transition-all cursor-pointer group",
                isActive
                    ? "bg-accent border-primary/20 shadow-sm"
                    : "bg-card border-border hover:border-primary/20 hover:bg-accent/50"
            )}
        >
            <div className="flex items-center gap-2 mb-3">
                {apps.map((app, i) => (
                    <div key={i} className="p-1.5 bg-background rounded-md border border-border shadow-sm text-muted-foreground group-hover:text-foreground transition-colors">
                        <AppIcon app={app} className="w-3.5 h-3.5" />
                    </div>
                ))}
            </div>

            <h3 className="font-medium text-sm leading-snug mb-2 text-foreground group-hover:text-primary transition-colors">
                {title}
            </h3>

            <div className="flex items-center justify-between">
                <div className="flex gap-1.5">
                    {tags.map(tag => (
                        <span key={tag} className="px-1.5 py-0.5 rounded-full bg-secondary text-[10px] font-medium text-secondary-foreground">
                            {tag}
                        </span>
                    ))}
                </div>
                <span className="text-[10px] text-muted-foreground font-medium">{time}</span>
            </div>
        </div>
    )
}
