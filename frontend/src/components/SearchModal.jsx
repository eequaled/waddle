import { Search, X, Calendar, Chrome, Slack, Music, Monitor } from 'lucide-react'

export function SearchModal({ isOpen, onClose }) {
    if (!isOpen) return null

    return (
        <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 bg-background/80 backdrop-blur-sm">
            <div className="w-full max-w-2xl bg-card border border-border rounded-xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-200">
                {/* Header / Input */}
                <div className="flex items-center p-4 border-b border-border gap-3">
                    <Search className="w-5 h-5 text-muted-foreground" />
                    <input
                        autoFocus
                        type="text"
                        placeholder="Search your memories..."
                        className="flex-1 bg-transparent outline-none text-lg placeholder:text-muted-foreground text-foreground"
                    />
                    <button onClick={onClose} className="p-1 hover:bg-accent rounded-md transition-colors">
                        <X className="w-5 h-5 text-muted-foreground" />
                    </button>
                </div>

                {/* Filters */}
                <div className="px-4 py-3 bg-muted/30 border-b border-border flex items-center gap-4 text-sm">
                    <div className="flex items-center gap-2 text-muted-foreground hover:text-foreground cursor-pointer transition-colors">
                        <Calendar className="w-4 h-4" />
                        <span>Any time</span>
                    </div>
                    <div className="h-4 w-px bg-border" />
                    <div className="flex items-center gap-2">
                        <span className="text-muted-foreground">Apps:</span>
                        <div className="flex gap-1">
                            <button className="p-1 rounded bg-accent text-accent-foreground"><Chrome className="w-3.5 h-3.5" /></button>
                            <button className="p-1 rounded hover:bg-accent text-muted-foreground"><Slack className="w-3.5 h-3.5" /></button>
                        </div>
                    </div>
                </div>

                {/* Results (Mock) */}
                <div className="max-h-[60vh] overflow-y-auto p-2">
                    <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground uppercase">Recent</div>

                    <div className="group flex items-center gap-3 p-3 rounded-lg hover:bg-accent cursor-pointer transition-colors">
                        <div className="p-2 rounded-md bg-background border border-border text-muted-foreground group-hover:text-primary group-hover:border-primary/30 transition-colors">
                            <Slack className="w-5 h-5" />
                        </div>
                        <div className="flex-1">
                            <h4 className="font-medium text-foreground">Discussing Japan trip budget</h4>
                            <p className="text-sm text-muted-foreground line-clamp-1">Sarah: "I found some flight options that fit our budget..."</p>
                        </div>
                        <span className="text-xs text-muted-foreground">Yesterday</span>
                    </div>

                    <div className="group flex items-center gap-3 p-3 rounded-lg hover:bg-accent cursor-pointer transition-colors">
                        <div className="p-2 rounded-md bg-background border border-border text-muted-foreground group-hover:text-primary group-hover:border-primary/30 transition-colors">
                            <Chrome className="w-5 h-5" />
                        </div>
                        <div className="flex-1">
                            <h4 className="font-medium text-foreground">Flight research on Expedia</h4>
                            <p className="text-sm text-muted-foreground line-clamp-1">Cheap Flights from LAX to KIX - Expedia</p>
                        </div>
                        <span className="text-xs text-muted-foreground">Last Week</span>
                    </div>
                </div>

                {/* Footer */}
                <div className="p-2 border-t border-border bg-muted/30 text-xs text-muted-foreground flex justify-between px-4">
                    <span><strong>Enter</strong> to select</span>
                    <span><strong>Esc</strong> to close</span>
                </div>
            </div>
        </div>
    )
}
