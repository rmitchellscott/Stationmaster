module IcsRruleHelper
  # Enhanced occurrence wrapper that preserves parent event reference
  EnhancedOccurrence = Struct.new(:occurrence, :parent_event) do
    # Delegate timing methods to the occurrence
    def start_time
      occurrence.start_time
    end

    def end_time
      occurrence.end_time
    end

    # Delegate event properties to the parent event
    def summary
      parent_event.summary
    end

    def description
      parent_event.description
    end

    def status
      parent_event.status
    end

    def rrule
      parent_event.rrule
    end

    # Check if this is an occurrence (always true for enhanced occurrences)
    def is_a?(klass)
      if klass == Icalendar::Recurrence::Occurrence
        true
      else
        super
      end
    end

    # Method missing to forward any other method calls to parent event
    def method_missing(method, *args, &block)
      if parent_event.respond_to?(method)
        parent_event.send(method, *args, &block)
      else
        super
      end
    end

    def respond_to_missing?(method, include_private = false)
      parent_event.respond_to?(method, include_private) || super
    end
  end

  def occurrences(event)
    return [event] unless event.rrule.present?

    # Calculate date range for recurring events
    start_date = recurring_event_start_date
    end_date = recurring_event_end_date

    Rails.logger.info "Expanding recurring event '#{event.summary}' between #{start_date} and #{end_date}"

    # Use icalendar gem's built-in method to expand recurring events
    begin
      result = event.occurrences_between(start_date, end_date)
      expanded_count = result&.count || 0
      Rails.logger.info "Recurring event '#{event.summary}' expanded to #{expanded_count} occurrences"

      if expanded_count > 0 && result.first
        begin
          Rails.logger.info "First occurrence: #{result.first.dtstart}"
          Rails.logger.info "Last occurrence: #{result.last.dtstart}" if result.count > 1
        rescue NoMethodError
          # Handle Occurrence objects that don't have dtstart
          Rails.logger.info "First occurrence: #{result.first.start_time}"
          Rails.logger.info "Last occurrence: #{result.last.start_time}" if result.count > 1
        end
      end

      # Wrap occurrences with enhanced occurrence objects that preserve parent event reference
      enhanced_result = result&.map { |occurrence| EnhancedOccurrence.new(occurrence, event) }

      enhanced_result.nil? ? [event] : enhanced_result  # Handle nil return value
    rescue => e
      Rails.logger.error "Error expanding recurring event '#{event.summary}': #{e.message}"
      Rails.logger.error "RRULE: #{event.rrule.first}"
      [event]  # Fallback to single event if expansion fails
    end
  end
end