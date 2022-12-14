#!/usr/bin/env ruby

api_key = ARGV[0]

if api_key.nil?
  puts "Usage: seed.rb <api_token>"
  exit 1
end

mappings = {
  'Settings' => rand(2..9),
  'Checkout' => rand(2..9),
  'Add to cart' => rand(2..8),
  'Search' => rand(2..12),
}

mapping_configs = {
  'Settings' => {
    duration: 900,
    label_count: 20,
    thread_count: 10,
  },
  'Checkout' => {
    duration: 1800,
    label_count: 10,
    thread_count: 20
  },
  'Add to cart' => {
    duration: 900,
    label_count: 3,
    thread_count: 30,
  },
  'Search' => {
    duration: 900,
    label_count: 5,
    thread_count: 100,
  },
}

while mappings.any? { |_, count| count >= 0 }
  remaining = mappings.select { |_, v| v >= 0 }
  current_iter = remaining.to_a[rand(remaining.size)].first

  puts "Processing #{current_iter}"

  config = mapping_configs[current_iter]
  filename = "tmp/#{current_iter.downcase.gsub(' ', '_')}#{mappings[current_iter]}.jtl"
  
  generate_cmd = "./scripts/generate_jtl_file.rb '#{filename}' #{config[:duration]} #{config[:label_count]} #{config[:thread_count]}"
  puts "Running generate: #{generate_cmd}"

  `#{generate_cmd}`

  publish_cmd = "go run main.go publish --env development --label '#{current_iter}' --file '#{filename}' --api-key #{api_key}"
  puts "Running publish: #{publish_cmd}"

  `#{publish_cmd}`

  mappings[current_iter] -= 1
end

`rm tmp/*.jtl`

puts "Done!"
