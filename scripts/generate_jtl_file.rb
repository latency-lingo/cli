#!/usr/bin/env ruby

require "csv"

class Label
  attr_accessor :name, :latency_range
end

# example
# timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,IdleTime,Connect
# 1650283530371,1072,SC18B_T00_OPSCashering_Launch,200,"Number of samples in transaction : 1, number of failing samples : 0",SC18B 18-1,,true,,10228,1037,1,19,null,0,2,794

output_filename = ARGV[0]
duration = ARGV[1]
label_count = ARGV[2]
thread_count = ARGV[3]
thread_groups = 1

if [output_filename, duration, label_count, thread_count].any?(&:nil?)
  puts "Usage: generate_jtl_file.rb <output_filename> <duration> <label_count> <thread_count>"
  exit 1
end

start_time = Time.now
end_time = Time.now + duration.to_i

possible_latency_ranges = [
  *5.times.map { (0..0.5) },
  *15.times.map { (0.5..2) },
  *10.times.map { (2..5) },
  *2.times.map { (5..10) },
  *2.times.map { (10..20) },
]

labels = label_count.to_i.times.map do |i|
  label = Label.new
  label.name = "label#{i}"
  label.latency_range = possible_latency_ranges.sample
  label
end

thread_timestamps = thread_count.to_i.times.map { start_time }
threads_finished = thread_count.to_i.times.map { false }
thread_label_idxs = thread_count.to_i.times.map { 0 }

CSV.open(output_filename, "wb") do |csv|
  csv << ["timeStamp", "elapsed", "label", "responseCode", "responseMessage", "threadName", "dataType", "success", "failureMessage", "bytes", "sentBytes", "grpThreads", "allThreads", "URL", "Latency", "IdleTime", "Connect"]

  while threads_finished.any? { |finished| finished == false }
    threads_finished.each_with_index do |finished, i|
      if finished == false
        # generate random latency and add random milliseconds
        label = labels[thread_label_idxs[i]]
        simulated_latency_sec = rand(label.latency_range) + rand
        simulated_latency_ms = (simulated_latency_sec * 1000).round
        thread_timestamps[i] += simulated_latency_sec

        # TODO(bobsin): introduce error rate
        # TODO(bobsin): introduce thread ramp up time
        csv << [(thread_timestamps[i].to_f * 1000).round, simulated_latency_ms, label.name, 200, "OK", "Thread-#{i}", "", true, "", rand(5000..10000), rand(500..1500), 1, thread_count, "", 0, 2, 794]

        if thread_label_idxs[i] == labels.length - 1
          thread_label_idxs[i] = 0
        else
          thread_label_idxs[i] += 1
        end

        if thread_timestamps[i] > end_time
          threads_finished[i] = true
        end
      end
    end
  end
end
